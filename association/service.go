package association

import (
	"context"
	"fmt"
	"runtime"
	"time"

	"github.com/edgestore/edgestore/internal/errors"
	"github.com/edgestore/edgestore/internal/eventstore"
	"github.com/edgestore/edgestore/internal/model"
	"github.com/edgestore/edgestore/internal/worker"
	"github.com/redis/go-redis/v9"
	"github.com/sirupsen/logrus"
)

func NewCacheKey(prefix string, id model.ID, tenantID model.ID) string {
	if prefix == "" {
		return fmt.Sprintf("%s:%s", tenantID, id)
	}

	return fmt.Sprintf("%s:%s:%s", prefix, tenantID, id)
}

// DefaultExpiration is set to never expire.
var DefaultExpiration = time.Duration(0)

var MaxWorkerSize = runtime.NumCPU()
var MaxQueueSize = MaxWorkerSize * 4

func NewSerializer() *eventstore.JSONSerializer {
	events := []model.Event{
		AssociationDeleted{},
		AssociationInserted{},
		AssociationUpdated{},
	}

	return eventstore.NewJSONSerializer(events...)
}

type Service struct {
	associations  *eventstore.Repository
	cache         *redis.Client
	cachePrefix   string
	jobDispatcher *worker.Dispatcher
	jobQueue      chan worker.Job
	logger        logrus.FieldLogger
}

type Config struct {
	Cache          *redis.Client
	CacheKeyPrefix string
	Logger         logrus.FieldLogger
	Observers      []eventstore.Observer
	Store          eventstore.Store
}

func New(cfg *Config) *Service {
	jobQueue := make(chan worker.Job, MaxQueueSize)
	dispatcher := worker.NewDispatcher(jobQueue, MaxWorkerSize, cfg.Logger)
	dispatcher.Run()

	return &Service{
		associations:  eventstore.NewRepository(&Association{}, cfg.Store, NewSerializer(), cfg.Logger, cfg.Observers...),
		cache:         cfg.Cache,
		cachePrefix:   cfg.CacheKeyPrefix,
		jobDispatcher: dispatcher,
		jobQueue:      jobQueue,
		logger:        cfg.Logger.WithField("component", "association-service"),
	}
}

func (s *Service) getAssociationFromCache(ctx context.Context, id model.ID, tenantID model.ID) (*Association, error) {
	key := NewCacheKey(s.cachePrefix, id, tenantID)

	m, err := s.cache.HGetAll(ctx, key).Result()
	if err != nil {
		return nil, err
	}

	if len(m) == 0 {
		return nil, errors.E(errors.NotFound, fmt.Sprintf("association %s not found in cache", key))
	}

	assoc, err := convertMapStringToAssociation(m)
	if err != nil {
		return nil, errors.E(err, errors.Internal, fmt.Sprintf("unable to parse cached association %s", key))
	}

	return assoc, nil
}

func (s *Service) setAssociationToCache(ctx context.Context, assoc *Association) error {
	assocKey := NewCacheKey(s.cachePrefix, assoc.ID, assoc.TenantID)

	m := convertAssociationToMapString(assoc)
	if _, err := s.cache.HMSet(ctx, assocKey, m).Result(); err != nil {
		return err
	}

	typeKey := NewCacheKey(s.cachePrefix, NewAssociationTypeID(assoc.In, assoc.Type), assoc.TenantID)
	z := redis.Z{
		Member: assocKey,
		Score:  float64(assoc.UpdatedAt.Unix()),
	}

	if _, err := s.cache.ZAdd(ctx, typeKey, z).Result(); err != nil {
		return err
	}

	return nil
}

func (s *Service) getAssociationFromDatabase(ctx context.Context, id model.ID, tenantID model.ID) (*Association, error) {
	agg, err := s.associations.Load(ctx, id, tenantID)
	if err != nil {
		return nil, err
	}

	return agg.(*Association), err
}

func (s *Service) applyAssociationToDatabase(ctx context.Context, cmd model.Command) (*Association, error) {
	// Create new aggregate
	if _, err := s.associations.Apply(ctx, cmd); err != nil {
		s.logger.Error(err)
		return nil, err
	}

	// Load aggregate
	agg, err := s.associations.Load(ctx, model.ID(cmd.CommandID()), cmd.CommandTenantID())
	if err != nil {
		return nil, err
	}

	return agg.(*Association), nil
}

func (s *Service) GetAssociation(ctx context.Context, id model.ID, tenantID model.ID) (*Association, error) {
	const op errors.Op = "graph/Service.GetAssociation"
	s.logger.Infof("%s: id=%s, tenant=%s", id, tenantID)

	cached, err := s.getAssociationFromCache(ctx, id, tenantID)
	if err != nil && !errors.Is(errors.NotFound, err) {
		return nil, err
	}

	if cached != nil {
		return cached, nil
	}

	// Cache miss
	assoc, err := s.getAssociationFromDatabase(ctx, id, tenantID)
	if err != nil {
		return nil, err
	}

	// Set aside cache
	s.jobQueue <- worker.NewJob(fmt.Sprintf("set-entity-cache-%s", assoc.ID), NewSetAssociationToCacheHandler(assoc, s))

	return assoc, nil
}

func (s *Service) CreateAssociation(ctx context.Context, cmd *InsertAssociation) error {
	const op errors.Op = "graph/Service.CreateAssociation"
	s.logger.Infof("%s: tenant=%s in=%s, out=%s, %atype=%s", op, cmd.TenantID, cmd.In, cmd.Out, cmd.Type)

	cmd.ID = NewAssociationID(cmd.In, cmd.Type, cmd.Out)
	old, err := s.GetAssociation(ctx, cmd.ID, cmd.TenantID)
	if err != nil && !errors.Is(errors.NotFound, err) {
		return err
	}

	if old != nil {
		return errors.E(op, errors.Duplicate, fmt.Sprintf("association %s already exists", cmd.ID))
	}

	s.jobQueue <- worker.NewJob(fmt.Sprintf("create-%s", cmd.ID), NewApplyAssociationHandler(cmd, s))

	return nil
}

func (s *Service) UpdateAssociation(ctx context.Context, cmd *UpdateAssociation) error {
	const op errors.Op = "graph/Service.UpdateAssociation"
	s.logger.Infof("%s: id=%s, tenant=%s", op, cmd.ID, cmd.TenantID)

	if _, err := s.GetAssociation(ctx, cmd.ID, cmd.TenantID); err != nil {
		return err
	}

	s.jobQueue <- worker.NewJob(fmt.Sprintf("update-%s", cmd.ID), NewApplyAssociationHandler(cmd, s))

	return nil
}

func (s *Service) DeleteAssociation(ctx context.Context, cmd *DeleteAssociation) error {
	const op errors.Op = "graph/Service.DeleteAssociation"
	s.logger.Infof("%s: id=%s, tenant=%s", op, cmd.ID, cmd.TenantID)

	if _, err := s.GetAssociation(ctx, cmd.ID, cmd.TenantID); err != nil {
		return err
	}

	s.jobQueue <- worker.NewJob(fmt.Sprintf("delete-%s", cmd.ID), NewApplyAssociationHandler(cmd, s))

	return nil
}

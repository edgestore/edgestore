package entity

import (
	"context"
	"fmt"
	"runtime"
	"time"

	"github.com/edgestore/edgestore/internal/errors"
	"github.com/edgestore/edgestore/internal/eventstore"
	"github.com/edgestore/edgestore/internal/model"
	"github.com/edgestore/edgestore/internal/worker"
	"github.com/go-redis/redis"
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
		EntityDeleted{},
		EntityInserted{},
		EntityUpdated{},
	}

	return eventstore.NewJSONSerializer(events...)
}

type Service struct {
	cache         *redis.Client
	cachePrefix   string
	entities      *eventstore.Repository
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
		cache:         cfg.Cache,
		cachePrefix:   cfg.CacheKeyPrefix,
		entities:      eventstore.NewRepository(&Entity{}, cfg.Store, NewSerializer(), cfg.Logger, cfg.Observers...),
		jobDispatcher: dispatcher,
		jobQueue:      jobQueue,
		logger:        cfg.Logger.WithField("component", "entity-service"),
	}
}

func (s *Service) getEntityFromCache(ctx context.Context, id model.ID, tenantID model.ID) (*Entity, error) {
	key := NewCacheKey(s.cachePrefix, id, tenantID)

	m, err := s.cache.WithContext(ctx).HGetAll(key).Result()
	if err != nil {
		return nil, err
	}

	if len(m) == 0 {
		return nil, errors.E(errors.NotFound, fmt.Sprintf("entity %s not found in cache", key))
	}

	entity, err := convertMapStringToEntity(m)
	if err != nil {
		return nil, errors.E(err, errors.Internal, fmt.Sprintf("unable to parse cached entity %s", key))
	}

	return entity, nil
}

func (s *Service) setEntityToCache(ctx context.Context, entity *Entity) error {
	key := NewCacheKey(s.cachePrefix, entity.ID, entity.TenantID)

	m := convertEntityToMap(entity)

	if _, err := s.cache.WithContext(ctx).HMSet(key, m).Result(); err != nil {
		return err
	}

	return nil
}

func (s *Service) getEntityFromDatabase(ctx context.Context, id model.ID, tenantID model.ID) (*Entity, error) {
	agg, err := s.entities.Load(ctx, id, tenantID)
	if err != nil {
		return nil, err
	}

	return agg.(*Entity), err
}

func (s *Service) applyEntityToDatabase(ctx context.Context, cmd model.Command) (*Entity, error) {
	// Create new aggregate
	if _, err := s.entities.Apply(ctx, cmd); err != nil {
		s.logger.Error(err)
		return nil, err
	}

	// Load aggregate
	agg, err := s.entities.Load(ctx, model.ID(cmd.CommandID()), cmd.CommandTenantID())
	if err != nil {
		return nil, err
	}

	return agg.(*Entity), nil
}

func (s *Service) GetEntity(ctx context.Context, id model.ID, tenantID model.ID) (*Entity, error) {
	const op errors.Op = "graph/Service.GetEntity"
	s.logger.Infof("%s: id=%s, tenant=%s", op, id, tenantID)

	if id == "" {
		return nil, errors.E(errors.Invalid, "ID is required")
	}

	if tenantID == "" {
		return nil, errors.E(errors.Invalid, "Tenant ID cannot be empty")
	}

	cached, err := s.getEntityFromCache(ctx, id, tenantID)
	if err != nil && !errors.Is(errors.NotFound, err) {
		return nil, err
	}

	if cached != nil {
		return cached, nil
	}

	// Cache miss
	entity, err := s.getEntityFromDatabase(ctx, id, tenantID)
	if err != nil {
		return nil, err
	}

	// Set aside cache
	s.jobQueue <- worker.NewJob(fmt.Sprintf("set-entity-cache-%s", NewCacheKey(s.cachePrefix, entity.ID, entity.TenantID)), NewSetEntityToCacheHandler(entity, s))

	return entity, nil
}

func (s *Service) CreateEntity(ctx context.Context, cmd *InsertEntity) error {
	const op errors.Op = "graph/Service.CreateEntity"
	s.logger.Infof("%s: id=%s, tenant=%s, type=%s", op, cmd.ID, cmd.TenantID, cmd.Type)

	old, err := s.GetEntity(ctx, cmd.ID, cmd.TenantID)
	if err != nil && !errors.Is(errors.NotFound, err) {
		return err
	}

	key := NewCacheKey(s.cachePrefix, cmd.ID, cmd.TenantID)
	if old != nil {
		return errors.E(op, errors.Duplicate, fmt.Sprintf("entity %s already exists", key))
	}

	s.jobQueue <- worker.NewJob(fmt.Sprintf("create-%s", key), NewApplyEntityHandler(cmd, s))

	return nil
}

func (s *Service) UpdateEntity(ctx context.Context, cmd *UpdateEntity) error {
	const op errors.Op = "graph/Service.UpdateEntity"
	s.logger.Infof("%s: id=%s, tenant=%s", op, cmd.ID, cmd.TenantID)

	if _, err := s.GetEntity(ctx, cmd.ID, cmd.TenantID); err != nil {
		return err
	}

	key := NewCacheKey(s.cachePrefix, cmd.ID, cmd.TenantID)
	s.jobQueue <- worker.NewJob(fmt.Sprintf("update-%s", key), NewApplyEntityHandler(cmd, s))

	return nil
}

func (s *Service) DeleteEntity(ctx context.Context, cmd *DeleteEntity) error {
	const op errors.Op = "graph/Service.DeleteEntity"
	s.logger.Infof("%s: id=%s, tenant=%s", op, cmd.ID, cmd.TenantID)

	if _, err := s.GetEntity(ctx, cmd.ID, cmd.TenantID); err != nil {
		return err
	}

	key := NewCacheKey(s.cachePrefix, cmd.ID, cmd.TenantID)
	s.jobQueue <- worker.NewJob(fmt.Sprintf("delete-%s", key), NewApplyEntityHandler(cmd, s))

	return nil
}

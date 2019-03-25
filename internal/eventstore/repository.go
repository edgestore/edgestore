package eventstore

import (
	"context"
	"reflect"
	"time"

	"github.com/edgestore/edgestore/internal/errors"
	"github.com/edgestore/edgestore/internal/model"
	"github.com/sirupsen/logrus"
)

type Aggregate interface {
	On(event model.Event) error
}

type Observer func(event model.Event)

// Repository provides the primary abstraction to saving and loading events.
type Repository struct {
	logger     logrus.FieldLogger
	observers  []Observer
	prototype  reflect.Type
	serializer Serializer
	store      Store
}

// New returns a new instance of the aggregate
func (r *Repository) NewAggregate() Aggregate {
	return reflect.New(r.prototype).Interface().(Aggregate)
}

// Save persists the events into the underlying Store
func (r *Repository) Save(ctx context.Context, tenantID model.ID, events ...model.Event) error {
	if len(events) == 0 {
		return nil
	}

	id := events[0].EventID()
	history := make(History, 0, len(events))
	for _, event := range events {
		record, err := r.serializer.MarshalEvent(event)
		if err != nil {
			return err
		}

		history = append(history, record)
	}

	return r.store.Save(ctx, id, tenantID, history)
}

// Load retrieves the specified aggregate from the underlying store
func (r *Repository) Load(ctx context.Context, aggregateID model.ID, tenantID model.ID) (Aggregate, error) {
	v, _, err := r.loadVersion(ctx, aggregateID, tenantID, 0)
	return v, err
}

// LoadVersion retrieves the specified aggregate from the underlying store at a particular version.
func (r *Repository) loadVersion(ctx context.Context, aggregateID model.ID, tenantID model.ID, version model.Version) (Aggregate, model.Version, error) {
	const op errors.Op = "store/Repository.loadVersion"
	history, err := r.store.Load(ctx, aggregateID, tenantID, 0, version)
	if err != nil {
		return nil, 0, err
	}

	count := len(history)
	if count == 0 {
		return nil, 0, errors.E(op, errors.NotFound)
	}

	aggregate := r.NewAggregate()

	r.logger.Debugf("loaded %d event(s) for %s", count, aggregateID)

	version = 0
	for _, record := range history {
		event, err := r.serializer.UnmarshalEvent(record)
		if err != nil {
			return nil, 0, err
		}

		if err := aggregate.On(event); err != nil {
			return nil, 0, err
		}

		version = event.EventVersion()
	}

	return aggregate, version, nil
}

// loadTime loads the specified aggregate from the store at some point in time and returns
// both the Aggregate and the current version number of the aggregate.
func (r *Repository) loadTime(ctx context.Context, aggregateID model.ID, tenantID model.ID, end time.Time) (Aggregate, model.Version, error) {
	const op errors.Op = "store/repository.loadTime"
	history, err := r.store.Load(ctx, aggregateID, tenantID, 0, 0)
	if err != nil {
		return nil, 0, err
	}

	count := len(history)
	if count == 0 {
		return nil, 0, errors.E(op, errors.NotFound)
	}

	aggregate := r.NewAggregate()

	r.logger.Debugf("loaded %d event(s) for %s", count, aggregateID)

	version := model.Version(0)
	for _, record := range history {
		event, err := r.serializer.UnmarshalEvent(record)
		if err != nil {
			return nil, 0, err
		}

		if event.EventAt().After(end) {
			break
		}

		if err := aggregate.On(event); err != nil {
			return nil, 0, err
		}

		version = event.EventVersion()
	}

	return aggregate, version, nil
}

// Apply executes the command specified and returns the current version of the aggregate
func (r *Repository) Apply(ctx context.Context, cmd model.Command) (model.Version, error) {
	const op errors.Op = "store/Repository.loadTime"

	if cmd == nil {
		return 0, errors.E(op, "command cannot be nil")
	}

	id := cmd.CommandID()
	if id == "" {
		return 0, errors.E(op, errors.Invalid, "required ID")
	}

	tenantID := cmd.CommandTenantID()
	if tenantID == "" {
		return 0, errors.E(op, errors.Invalid, "required tenant ID")
	}

	aggregate, version, err := r.loadVersion(ctx, id, tenantID, 0)
	if err != nil {
		aggregate = r.NewAggregate()
	}

	h, ok := aggregate.(model.CommandHandler)
	if !ok {
		return 0, errors.E(op, errors.Internal, "aggregate %v, does not implement CommandHandler")
	}

	events, err := h.Apply(ctx, cmd)
	if err != nil {
		return 0, err
	}

	err = r.Save(ctx, tenantID, events...)
	if err != nil {
		return 0, err
	}

	totalEvents := len(events)
	if v := totalEvents; v > 0 {
		version = events[v-1].EventVersion()
	}

	// publish events to observers
	if r.observers != nil {
		for _, event := range events {
			for _, observer := range r.observers {
				observer(event)
			}
		}
	}

	r.logger.Debugf("applied %d event(s)", totalEvents)

	return version, nil
}

func (r *Repository) Store() Store {
	return r.store
}

func (r *Repository) Serializer() Serializer {
	return r.serializer
}

func NewRepository(prototype Aggregate, store Store, serializer Serializer, logger logrus.FieldLogger, observers ...Observer) *Repository {
	t := reflect.TypeOf(prototype)
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}

	return &Repository{
		prototype:  t,
		store:      store,
		observers:  observers,
		serializer: serializer,
		logger:     logger.WithField("component", "repository"),
	}
}

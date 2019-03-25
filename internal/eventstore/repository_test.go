package eventstore

import (
	"context"
	"testing"
	"time"

	"github.com/edgestore/edgestore/internal/errors"
	"github.com/edgestore/edgestore/internal/model"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

type Entity struct {
	ID        model.ID
	TenantID  model.ID
	Version   model.Version
	Name      string
	CreatedAt *time.Time
	UpdatedAt *time.Time
}

type EntityCreated struct {
	model.EventModel
}

type EntityNameSet struct {
	model.EventModel
	Name string
}

func (item *Entity) On(event model.Event) error {
	switch v := event.(type) {
	case *EntityCreated:
		item.Version = v.Version
		item.ID = v.ID
		item.TenantID = v.TenantID
		item.CreatedAt = v.At
		item.UpdatedAt = v.At

	case *EntityNameSet:
		item.Version = v.EventModel.Version
		item.Name = v.Name
		item.UpdatedAt = v.EventModel.At

	default:
		return errors.E(errors.Invalid)
	}

	return nil
}

type CreateEntity struct {
	model.CommandModel
}

type Nop struct {
	model.CommandModel
}

func (item *Entity) Apply(ctx context.Context, command model.Command) ([]model.Event, error) {
	now := time.Now()
	switch command.(type) {
	case *CreateEntity:
		return []model.Event{&EntityCreated{
			EventModel: model.EventModel{
				ID:       command.CommandID(),
				TenantID: command.CommandTenantID(),
				Version:  item.Version + 1,
				At:       &now,
			},
		}}, nil

	case *Nop:
		return []model.Event{}, nil

	default:
		return []model.Event{}, nil
	}
}

func TestNew(t *testing.T) {
	logger := logrus.New()
	repository := NewRepository(&Entity{}, NewInMemory(logger), NewJSONSerializer(), logger)
	aggregate := repository.NewAggregate()
	assert.NotNil(t, aggregate)
	assert.Equal(t, &Entity{}, aggregate)
}

func TestRepository_Load_NotFound(t *testing.T) {
	ctx := context.Background()
	logger := logrus.New()
	repository := NewRepository(&Entity{}, NewInMemory(logger), NewJSONSerializer(), logger)

	_, err := repository.Load(ctx, "does-not-exist", "anonymous")
	assert.NotNil(t, err)
	assert.True(t, errors.Is(errors.NotFound, err))
}

func TestRegistry(t *testing.T) {
	ctx := context.Background()
	id := model.ID("123")
	tenantID := model.ID("anonymous")
	name := "Jones"
	serializer := NewJSONSerializer(
		EntityCreated{},
		EntityNameSet{},
	)

	t.Run("simple", func(t *testing.T) {
		logger := logrus.New()
		repository := NewRepository(&Entity{}, NewInMemory(logger), serializer, logger)

		// Test - Add an event to the store and verify we can recreate the entity

		start := time.Unix(3, 0)
		end := time.Unix(4, 0)
		err := repository.Save(ctx, tenantID,
			&EntityCreated{
				EventModel: model.EventModel{ID: id, TenantID: tenantID, Version: 0, At: &start},
			},
			&EntityNameSet{
				EventModel: model.EventModel{ID: id, TenantID: tenantID, Version: 1, At: &end},
				Name:       name,
			},
		)
		assert.Nil(t, err)

		v, err := repository.Load(ctx, id, tenantID)
		assert.Nil(t, err, "expected successful load")

		org, ok := v.(*Entity)
		assert.True(t, ok)
		assert.Equal(t, id, org.ID, "expected restored id")
		assert.Equal(t, name, org.Name, "expected restored name")

		// Test - Update the org name and verify that the change is reflected in the loaded result

		updated := "Sarah"
		err = repository.Save(ctx, tenantID, &EntityNameSet{
			EventModel: model.EventModel{ID: id, TenantID: tenantID, Version: 2},
			Name:       updated,
		})
		assert.Nil(t, err)

		v, err = repository.Load(ctx, id, tenantID)
		assert.Nil(t, err)

		org, ok = v.(*Entity)
		assert.True(t, ok)
		assert.Equal(t, id, org.ID)
		assert.Equal(t, updated, org.Name)
	})

	t.Run("with pointer prototype", func(t *testing.T) {
		logger := logrus.New()
		registry := NewRepository(&Entity{}, NewInMemory(logger), serializer, logger)
		start := time.Unix(3, 0)
		end := time.Unix(4, 0)

		err := registry.Save(ctx, tenantID,
			&EntityCreated{
				EventModel: model.EventModel{ID: id, TenantID: tenantID, Version: 0, At: &start},
			},
			&EntityNameSet{
				EventModel: model.EventModel{ID: id, TenantID: tenantID, Version: 1, At: &end},
				Name:       name,
			},
		)
		assert.Nil(t, err)

		v, err := registry.Load(ctx, id, tenantID)
		assert.Nil(t, err)
		assert.Equal(t, name, v.(*Entity).Name)
	})

	t.Run("with pointer bind", func(t *testing.T) {
		logger := logrus.New()
		registry := NewRepository(&Entity{}, NewInMemory(logger), serializer, logger)
		start := time.Unix(3, 0)
		err := registry.Save(ctx, tenantID,
			&EntityNameSet{
				EventModel: model.EventModel{ID: id, TenantID: tenantID, Version: 0, At: &start},
				Name:       name,
			},
		)
		assert.Nil(t, err)

		v, err := registry.Load(ctx, id, tenantID)
		assert.Nil(t, err)
		assert.Equal(t, name, v.(*Entity).Name)
	})
}

func TestAt(t *testing.T) {
	ctx := context.Background()
	id := model.ID("123")
	tenantID := model.ID("anonymous")

	serializer := NewJSONSerializer(EntityCreated{})
	logger := logrus.New()
	registry := NewRepository(&Entity{}, NewInMemory(logger), serializer, logger)

	now := time.Now()
	err := registry.Save(ctx, tenantID,
		&EntityCreated{
			EventModel: model.EventModel{ID: id, TenantID: tenantID, Version: 1, At: &now},
		},
	)
	assert.Nil(t, err)

	v, err := registry.Load(ctx, id, tenantID)
	assert.Nil(t, err)

	org := v.(*Entity)
	assert.NotZero(t, org.CreatedAt)
	assert.NotZero(t, org.UpdatedAt)
}

func TestRepository_SaveNostore_Events(t *testing.T) {
	logger := logrus.New()
	repository := NewRepository(&Entity{}, NewInMemory(logger), NewJSONSerializer(), logger)
	err := repository.Save(context.Background(), model.ID(""))
	assert.Nil(t, err)
}

func TestWithObservers(t *testing.T) {
	captured := []model.Event{}
	observer := func(event model.Event) {
		captured = append(captured, event)
	}

	serializer := NewJSONSerializer(
		EntityCreated{},
		EntityNameSet{},
	)

	logger := logrus.New()
	repository := NewRepository(&Entity{}, NewInMemory(logger), serializer, logger, observer)

	ctx := context.Background()

	// When I apply command
	_, err := repository.Apply(ctx, &CreateEntity{
		CommandModel: model.CommandModel{ID: "abc", TenantID: "anonymous"},
	})

	// Then I expect event to be captured
	assert.Nil(t, err)
	assert.Len(t, captured, 1)

	_, ok := captured[0].(*EntityCreated)
	assert.True(t, ok)
}

func TestApply(t *testing.T) {
	serializer := NewJSONSerializer(
		EntityCreated{},
	)

	logger := logrus.New()
	repo := NewRepository(&Entity{}, NewInMemory(logger), serializer, logger)

	cmd := &CreateEntity{CommandModel: model.CommandModel{ID: "123", TenantID: "anonymous"}}

	// When
	version, err := repo.Apply(context.Background(), cmd)

	// Then
	assert.Nil(t, err)
	assert.EqualValues(t, 1, version)

	// And
	version, err = repo.Apply(context.Background(), cmd)

	// Then
	assert.Nil(t, err)
	assert.EqualValues(t, 2, version)
}

func TestApplyNopCommand(t *testing.T) {
	t.Run("Version still returned when command generates no events", func(t *testing.T) {
		repo := NewRepository(&Entity{}, NewInMemory(logrus.New()), NewJSONSerializer(
			EntityCreated{},
		), logrus.New())

		cmd := &Nop{
			CommandModel: model.CommandModel{ID: "abc", TenantID: "anonymous"},
		}
		version, err := repo.Apply(context.Background(), cmd)
		assert.Nil(t, err)
		assert.EqualValues(t, 0, version)
	})
}

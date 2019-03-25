package entity

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/edgestore/edgestore/internal/model"

	"github.com/edgestore/edgestore/internal/errors"
)

type Entity struct {
	CreatedAt *time.Time    `json:"created_at"`
	DeletedAt *time.Time    `json:"deleted_at,omitempty"`
	ID        model.ID      `json:"id"`
	Data      model.Data    `json:"data,omitempty"`
	TenantID  model.ID      `json:"tenant_id"`
	Type      string        `json:"otype"`
	UpdatedAt *time.Time    `json:"updated_at"`
	Version   model.Version `json:"version"`
}

type InsertEntity struct {
	model.CommandModel
	Data model.Data `json:"data"`
	Type string     `json:"otype" binding:"required"`
}

type UpdateEntity struct {
	model.CommandModel
	Data model.Data `json:"data"`
}

type DeleteEntity struct {
	model.CommandModel
}

type EntityInserted struct {
	model.EventModel
	Data model.Data `json:"data"`
	Type string     `json:"otype"`
}

type EntityUpdated struct {
	model.EventModel
	Data model.Data `json:"data"`
}

type EntityDeleted struct {
	model.EventModel
	DeletedAt *time.Time
}

func (e *Entity) On(event model.Event) error {
	const op errors.Op = "graph/Entity.On"

	switch v := event.(type) {
	case *EntityInserted:
		e.Data = v.Data
		e.Type = v.Type
	case *EntityUpdated:
		e.Data = v.Data
	case *EntityDeleted:
		e.DeletedAt = v.DeletedAt
	default:
		return errors.E(op, errors.Internal, fmt.Errorf("invalid event %T", event))
	}

	e.ID = event.EventID()
	e.TenantID = event.EventTenantID()
	e.Version = event.EventVersion()

	if int64(e.Version) == 1 {
		e.CreatedAt = event.EventAt()
	}

	e.UpdatedAt = event.EventAt()

	return nil
}

func (e *Entity) Apply(ctx context.Context, cmd model.Command) ([]model.Event, error) {
	const op errors.Op = "graph/Entity.Apply"

	if cmd.CommandID() == "" {
		return nil, errors.E(op, errors.Internal, "missing ID")
	}

	if cmd.CommandTenantID() == "" {
		return nil, errors.E(op, errors.Internal, "missing tenant ID")
	}

	var events []model.Event
	switch v := cmd.(type) {
	case *InsertEntity:
		inserted, err := e.applyInsert(v)
		if err != nil {
			return nil, errors.E(op, err)
		}

		events = append(events, inserted)
	case *UpdateEntity:
		updated, err := e.applyUpdate(v)
		if err != nil {
			return nil, errors.E(op, err)
		}

		events = append(events, updated)
	case *DeleteEntity:
		deleted, err := e.applyDelete(v)
		if err != nil {
			return nil, errors.E(op, err)
		}

		events = append(events, deleted)
	default:
		return nil, errors.E(op, errors.Internal, "unknown command")
	}

	return events, nil
}

func (e *Entity) applyInsert(cmd *InsertEntity) (model.Event, error) {
	if cmd.Type == "" {
		return nil, errors.E(errors.Invalid, "missing type")
	}

	now := time.Now()
	inserted := &EntityInserted{
		EventModel: model.EventModel{
			ID:       cmd.CommandID(),
			TenantID: cmd.CommandTenantID(),
			Version:  e.Version + 1,
			At:       &now,
		},
		Data: cmd.Data,
		Type: cmd.Type,
	}

	return inserted, nil
}

func (e *Entity) applyUpdate(cmd *UpdateEntity) (model.Event, error) {
	now := time.Now()
	updated := &EntityUpdated{
		EventModel: model.EventModel{
			ID:       cmd.CommandID(),
			TenantID: cmd.CommandTenantID(),
			Version:  e.Version + 1,
			At:       &now,
		},
		Data: cmd.Data,
	}

	return updated, nil
}

func (e *Entity) applyDelete(cmd *DeleteEntity) (model.Event, error) {
	now := time.Now()
	deleted := &EntityDeleted{
		EventModel: model.EventModel{
			ID:       cmd.CommandID(),
			TenantID: cmd.CommandTenantID(),
			Version:  e.Version + 1,
			At:       &now,
		},
		DeletedAt: &now,
	}

	return deleted, nil
}

func convertMapStringToEntity(m map[string]string) (*Entity, error) {
	var createdAt *time.Time
	if _, exists := m["created_at"]; exists {
		value, err := time.Parse(time.RFC3339, m["created_at"])
		if err != nil {
			return nil, err
		}

		createdAt = &value
	}

	var deletedAt *time.Time
	if _, exists := m["deleted_at"]; exists {
		t, err := time.Parse(time.RFC3339, m["deleted_at"])
		if err != nil {
			return nil, err
		}

		deletedAt = &t
	}

	var updatedAt *time.Time
	if _, exists := m["updated_at"]; exists {
		t, err := time.Parse(time.RFC3339, m["updated_at"])
		if err != nil {
			return nil, err
		}

		updatedAt = &t
	}

	var data model.Data
	if _, exists := m["data"]; exists {
		raw := []byte(m["data"])
		if err := json.Unmarshal(raw, &data); err != nil {
			return nil, err
		}
	}

	version, err := strconv.Atoi(m["version"])
	if err != nil {
		return nil, err
	}

	entity := &Entity{
		CreatedAt: createdAt,
		DeletedAt: deletedAt,
		ID:        model.ID(m["id"]),
		Data:      data,
		TenantID:  model.ID(m["tenant_id"]),
		Type:      m["otype"],
		UpdatedAt: updatedAt,
		Version:   model.Version(version),
	}

	return entity, nil
}

func convertEntityToMap(e *Entity) map[string]interface{} {
	m := make(map[string]interface{})

	if e.CreatedAt != nil {
		m["created_at"] = e.CreatedAt.Format(time.RFC3339)
	}

	if e.DeletedAt != nil {
		m["deleted_at"] = e.DeletedAt.Format(time.RFC3339)
	}

	if e.UpdatedAt != nil {
		m["updated_at"] = e.UpdatedAt.Format(time.RFC3339)
	}

	m["id"] = string(e.ID)

	if e.Data != nil {
		raw, err := json.Marshal(e.Data)
		if err != nil {
			panic(err)
		}

		m["data"] = string(raw)
	}

	m["tenant_id"] = string(e.TenantID)

	m["otype"] = string(e.Type)

	m["version"] = strconv.Itoa(int(e.Version))

	return m
}

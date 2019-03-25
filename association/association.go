package association

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/edgestore/edgestore/internal/errors"
	"github.com/edgestore/edgestore/internal/model"
)

func NewAssociationID(in model.ID, atype string, out model.ID) model.ID {
	return model.ID(fmt.Sprintf("%s:%s:%s", in, atype, out))
}

func NewAssociationTypeID(in model.ID, atype string) model.ID {
	return model.ID(fmt.Sprintf("%s:%s", in, atype))
}

type Association struct {
	CreatedAt *time.Time    `json:"created_at"`
	Data      model.Data    `json:"data,omitempty"`
	DeletedAt *time.Time    `json:"deleted_at,omitempty"`
	ID        model.ID      `json:"id"`
	In        model.ID      `json:"in"`
	Out       model.ID      `json:"out"`
	TenantID  model.ID      `json:"tenant_id"`
	Type      string        `json:"atype"`
	UpdatedAt *time.Time    `json:"updated_at"`
	Version   model.Version `json:"version"`
}

type InsertAssociation struct {
	model.CommandModel
	Data model.Data `json:"data"`
	In   model.ID   `json:"in" binding:"required"`
	Out  model.ID   `json:"out" binding:"required"`
	Type string     `json:"atype" binding:"required"`
}

type UpdateAssociation struct {
	model.CommandModel
	Data model.Data `json:"data"`
}

type DeleteAssociation struct {
	model.CommandModel
}

type AssociationInserted struct {
	model.EventModel
	Data model.Data `json:"data"`
	In   model.ID   `json:"in"`
	Out  model.ID   `json:"out"`
	Type string     `json:"atype"`
}

type AssociationUpdated struct {
	model.EventModel
	Data model.Data `json:"data"`
	Type string     `json:"atype"`
}

type AssociationDeleted struct {
	model.EventModel
	DeletedAt *time.Time
}

func (o *Association) On(event model.Event) error {
	const op errors.Op = "graph/Association.On"

	switch v := event.(type) {
	case *AssociationInserted:
		o.In = v.In
		o.Out = v.Out
		o.Data = v.Data
		o.Type = v.Type
	case *AssociationUpdated:
		o.Data = v.Data
		o.Type = v.Type
	case *AssociationDeleted:
		o.DeletedAt = v.DeletedAt
	default:
		return errors.E(op, errors.Internal, fmt.Errorf("invalid event %T", event))
	}

	o.ID = event.EventID()
	o.TenantID = event.EventTenantID()
	o.Version = event.EventVersion()

	if int64(o.Version) == 1 {
		o.CreatedAt = event.EventAt()
	}

	o.UpdatedAt = event.EventAt()

	return nil
}

func (o *Association) Apply(ctx context.Context, cmd model.Command) ([]model.Event, error) {
	const op errors.Op = "graph/Association.Apply"

	if cmd.CommandID() == "" {
		return nil, errors.E(op, errors.Internal, "missing ID")
	}

	if cmd.CommandTenantID() == "" {
		return nil, errors.E(op, errors.Internal, "missing tenant ID")
	}

	var events []model.Event
	switch v := cmd.(type) {
	case *InsertAssociation:
		inserted, err := o.applyInsert(v)
		if err != nil {
			return nil, errors.E(op, err)
		}

		events = append(events, inserted)
	case *UpdateAssociation:
		updated, err := o.applyUpdate(v)
		if err != nil {
			return nil, errors.E(op, err)
		}

		events = append(events, updated)
	case *DeleteAssociation:
		deleted, err := o.applyDelete(v)
		if err != nil {
			return nil, errors.E(op, err)
		}

		events = append(events, deleted)
	default:
		return nil, errors.E(op, errors.Internal, "unknown command")
	}

	return events, nil
}

func (o *Association) applyInsert(cmd *InsertAssociation) (model.Event, error) {
	// Validate required params
	if cmd.In == "" {
		return nil, errors.E(errors.Invalid, "missing input ID")
	}

	if cmd.Out == "" {
		return nil, errors.E(errors.Invalid, "missing output ID")
	}

	if cmd.Type == "" {
		return nil, errors.E(errors.Invalid, "missing type")
	}

	now := time.Now()
	inserted := &AssociationInserted{
		EventModel: model.EventModel{
			ID:       cmd.CommandID(),
			TenantID: cmd.CommandTenantID(),
			Version:  o.Version + 1,
			At:       &now,
		},
		In:   cmd.In,
		Out:  cmd.Out,
		Data: cmd.Data,
		Type: cmd.Type,
	}

	return inserted, nil
}

func (o *Association) applyUpdate(cmd *UpdateAssociation) (model.Event, error) {
	now := time.Now()
	updated := &AssociationUpdated{
		EventModel: model.EventModel{
			ID:       cmd.CommandID(),
			TenantID: cmd.CommandTenantID(),
			Version:  o.Version + 1,
			At:       &now,
		},
		Data: cmd.Data,
	}

	return updated, nil
}

func (o *Association) applyDelete(cmd *DeleteAssociation) (model.Event, error) {
	now := time.Now()
	deleted := &AssociationDeleted{
		EventModel: model.EventModel{
			ID:       cmd.CommandID(),
			TenantID: cmd.CommandTenantID(),
			Version:  o.Version + 1,
			At:       &now,
		},
		DeletedAt: &now,
	}

	return deleted, nil
}

func convertMapStringToAssociation(m map[string]string) (*Association, error) {
	var createdAt time.Time
	if _, exists := m["created_at"]; exists {
		value, err := time.Parse(time.RFC3339, m["created_at"])
		if err != nil {
			return nil, err
		}

		createdAt = value
	}

	var deletedAt time.Time
	if _, exists := m["deleted_at"]; exists {
		t, err := time.Parse(time.RFC3339, m["deleted_at"])
		if err != nil {
			return nil, err
		}

		deletedAt = t
	}

	var updatedAt time.Time
	if _, exists := m["updated_at"]; exists {
		t, err := time.Parse(time.RFC3339, m["updated_at"])
		if err != nil {
			return nil, err
		}

		updatedAt = t
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

	assoc := &Association{
		CreatedAt: &createdAt,
		DeletedAt: &deletedAt,
		ID:        model.ID(m["id"]),
		In:        model.ID(m["in"]),
		Out:       model.ID(m["out"]),
		Data:      data,
		TenantID:  model.ID(m["tenant_id"]),
		Type:      m["atype"],
		UpdatedAt: &updatedAt,
		Version:   model.Version(version),
	}

	return assoc, nil
}

func convertAssociationToMapString(a *Association) model.Data {
	m := make(model.Data)

	if a.CreatedAt != nil {
		m["created_at"] = a.CreatedAt.Format(time.RFC3339)
	}

	if a.DeletedAt != nil {
		m["deleted_at"] = a.DeletedAt.Format(time.RFC3339)
	}

	if a.UpdatedAt != nil {
		m["updated_at"] = a.UpdatedAt.Format(time.RFC3339)
	}

	m["id"] = string(a.ID)

	m["in"] = string(a.In)

	m["out"] = string(a.Out)

	if a.Data != nil {
		data, err := json.Marshal(a.Data)
		if err != nil {
			panic(err)
		}

		m["data"] = string(data)
	}

	m["tenant_id"] = string(a.TenantID)

	m["atype"] = string(a.Type)

	m["version"] = strconv.Itoa(int(a.Version))

	return m
}

package eventstore

import (
	"encoding/json"
	"fmt"
	"reflect"

	"github.com/edgestore/edgestore/internal/errors"
	"github.com/edgestore/edgestore/internal/model"
)

// Serializer converts between Events and Records
type Serializer interface {
	// MarshalEvent converts an Event to a Record
	MarshalEvent(event model.Event) (*Record, error)

	// UnmarshalEvent converts an Event backed into a Record
	UnmarshalEvent(record *Record) (model.Event, error)
}

type jsonEvent struct {
	Kind    string          `json:"kind"`
	Payload json.RawMessage `json:"payload"`
}

// JSONSerializer provides a simple serializer implementation
type JSONSerializer struct {
	eventTypes map[string]reflect.Type
}

// Bind registers the specified events with the serializer; may be called more than once
func (j *JSONSerializer) Bind(events ...model.Event) {
	for _, event := range events {
		eventType, t := model.EventType(event)
		j.eventTypes[eventType] = t
	}
}

// MarshalEvent converts an event into its persistent type, Record
func (j *JSONSerializer) MarshalEvent(event model.Event) (*Record, error) {
	const op errors.Op = "store/JSONSerializer.MarshalEvent"

	payload, err := json.Marshal(event)
	if err != nil {
		return nil, errors.E(op, errors.Internal, err, "unable to encode event")
	}

	eventType, _ := model.EventType(event)
	data, err := json.Marshal(jsonEvent{
		Kind:    eventType,
		Payload: json.RawMessage(payload),
	})
	if err != nil {
		return nil, errors.E(op, errors.Internal, err, "unable to encode JSON event")
	}

	return &Record{
		AggregateID: event.EventID(),
		TenantID:    event.EventTenantID(),
		Version:     event.EventVersion(),
		Data:        data,
	}, nil
}

// UnmarshalEvent converts the persistent type, Record, into an Event instance
func (j *JSONSerializer) UnmarshalEvent(record *Record) (model.Event, error) {
	const op errors.Op = "store/JSONSerializer.UnmarshalEvent"
	var wrapper jsonEvent
	err := json.Unmarshal(record.Data, &wrapper)
	if err != nil {
		return nil, errors.E(op, errors.Internal, err, "unable to decode JSON event")
	}

	t, ok := j.eventTypes[wrapper.Kind]
	if !ok {
		return nil, errors.E(op, errors.Internal, err, fmt.Sprintf("unbound event type %v", wrapper.Kind))
	}

	v := reflect.New(t).Interface()
	err = json.Unmarshal(wrapper.Payload, v)
	if err != nil {
		return nil, errors.E(op, errors.Internal, err, fmt.Sprintf("unable to decode event payload into %#v", v))
	}

	return v.(model.Event), nil
}

// MarshalAll is a utility that marshals all the events provided into a History entity
func (j *JSONSerializer) MarshalAll(events ...model.Event) (History, error) {
	history := make(History, 0, len(events))

	for _, event := range events {
		record, err := j.MarshalEvent(event)
		if err != nil {
			return nil, err
		}
		history = append(history, record)
	}

	return history, nil
}

// NewJSONSerializer constructs a new JSONSerializer and populates it with the specified events.
// Bind may be subsequently called to add more events.
func NewJSONSerializer(events ...model.Event) *JSONSerializer {
	serializer := &JSONSerializer{
		eventTypes: map[string]reflect.Type{},
	}
	serializer.Bind(events...)

	return serializer
}

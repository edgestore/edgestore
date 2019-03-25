package model

import (
	"reflect"
	"time"
)

// Event describe a change that happened to the aggregate
type Event interface {
	EventID() ID
	EventTenantID() ID
	EventVersion() Version
	EventAt() *time.Time
}

// EventTyper is an optional interface that an Event can implement that allows it
// to specify an event type different than the name of the struct
type EventTyper interface {
	// EventType returns the name of event type
	EventType() string
}

// EventModel provides a default implementation of an Event
type EventModel struct {
	// ID contains the aggregate ID.
	ID ID `json:"id"`

	// TenantID is the of the owner of an event.
	TenantID ID `json:"tenant_id"`

	// Version is the incremental version of an event
	Version Version `json:"version"`

	// At is the date the event was created
	At *time.Time `json:"at"`
}

func (m EventModel) EventID() ID {
	return m.ID
}

func (m EventModel) EventTenantID() ID {
	return m.TenantID
}

func (m EventModel) EventVersion() Version {
	return m.Version
}

func (m EventModel) EventAt() *time.Time {
	return m.At
}

// EventType is a helper func that extracts the event type of the event along with the reflect.Kind of the event.
//
// Primarily useful for serializers that need to understand how marshal and unmarshal instances of Event to a []byte
func EventType(event Event) (string, reflect.Type) {
	t := reflect.TypeOf(event)
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}

	if v, ok := event.(EventTyper); ok {
		return v.EventType(), t
	}

	return t.Name(), t
}

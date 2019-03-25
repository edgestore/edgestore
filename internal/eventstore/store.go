package eventstore

import (
	"context"
	"time"

	"github.com/edgestore/edgestore/internal/model"
)

// Record provides the serialized representation of the event
type Record struct {
	ID model.ID

	AggregateID model.ID

	TenantID model.ID

	// Version contains the version associated with the serialized event
	Version model.Version

	// Payload contains the event in serialized form
	Data []byte

	CreatedAt time.Time
}

// History represents
type History []*Record

// Len implements sort.Interface
func (h History) Len() int {
	return len(h)
}

// Swap implements s ort.Interface
func (h History) Swap(i, j int) {
	h[i], h[j] = h[j], h[i]
}

// Less implements sort.Interface
func (h History) Less(i, j int) bool {
	return h[i].Version < h[j].Version
}

// Store provides an abstraction for a repository
type Store interface {
	// Load the history of events up to the version specified.
	// When toVersion is 0, all events will be loaded.
	// To start at the beginning, fromVersion should be set to 0
	Load(ctx context.Context, aggregateID model.ID, tenantID model.ID, fromVersion, toVersion model.Version) (History, error)

	// Save the provided serialized records to the store
	Save(ctx context.Context, aggregateID model.ID, tenantID model.ID, records []*Record) error
}

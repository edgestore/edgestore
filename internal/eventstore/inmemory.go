package eventstore

import (
	"context"
	"sort"
	"sync"

	"github.com/edgestore/edgestore/internal/errors"
	"github.com/edgestore/edgestore/internal/model"
	"github.com/sirupsen/logrus"
)

type InMemory struct {
	mux    *sync.Mutex
	events map[model.ID]History

	logger logrus.FieldLogger
}

func NewInMemory(logger logrus.FieldLogger) Store {
	logger.Infof("InMemory Store")

	return &InMemory{
		mux:    &sync.Mutex{},
		events: map[model.ID]History{},
		logger: logger.WithField("component", "in-memory"),
	}
}

// Load implements the Store interface and retrieve events from In-Memory store
func (m *InMemory) Load(ctx context.Context, aggregateID model.ID, tenantID model.ID, fromVersion, toVersion model.Version) (History, error) {
	const op errors.Op = "persistence/InMemory.Load"
	m.logger.Debugf("load aggregate %s from tenant %s", aggregateID, tenantID)

	m.mux.Lock()
	defer m.mux.Unlock()

	records, ok := m.events[aggregateID+tenantID]
	if !ok {
		return nil, errors.E(op, errors.NotFound)
	}

	history := make(History, 0, len(records))
	if len(records) > 0 {
		for _, r := range records {
			if v := r.Version; v >= fromVersion && (toVersion == 0 || v <= toVersion) {
				history = append(history, r)
			}
		}
	}
	return records, nil
}

func (m *InMemory) Save(ctx context.Context, aggregateID model.ID, tenantID model.ID, records []*Record) error {
	m.logger.Debugf("save aggregate %s from tenant %s", aggregateID, tenantID)

	m.mux.Lock()
	defer m.mux.Unlock()

	history, ok := m.events[aggregateID+tenantID]
	if !ok {
		history = History{}
	}

	history = append(history, records...)
	sort.Sort(history)
	m.events[aggregateID+tenantID] = history

	return nil
}

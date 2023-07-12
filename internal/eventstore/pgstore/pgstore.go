package pgstore

import (
	"context"
	"fmt"
	"math"
	"reflect"
	"sort"
	"strings"

	"github.com/edgestore/edgestore/internal/errors"
	"github.com/edgestore/edgestore/internal/eventstore"
	"github.com/edgestore/edgestore/internal/model"
	"github.com/go-pg/pg/v10"
	"github.com/sirupsen/logrus"
)

var (
	selectMaxVersionSQL = "SELECT MAX(version) FROM records WHERE aggregate_id = ?aggregateID"
	selectRecordsSQL    = strings.TrimSpace(`
		SELECT id, aggregate_id, tenant_id, version, data, created_at FROM records
		WHERE aggregate_id = ?aggregateID AND tenant_id = ?tenantID AND version >= ?fromVersion AND version <= ?toVersion
		ORDER BY version ASC
	`)
)

type PgStore struct {
	tableName string
	db        *pg.DB
	logger    logrus.FieldLogger
}

// Load the history of events from PgStore, up to the version specified.
// When toVersion is 0, all events will be loaded.
// To start at the beginning, fromVersion should be set to 0
func (p *PgStore) Load(ctx context.Context, aggregateID model.ID, tenantID model.ID, fromVersion, toVersion model.Version) (eventstore.History, error) {
	const op errors.Op = "pgstore/PgStore.Load"

	if toVersion == 0 {
		toVersion = math.MaxInt32
	}

	history := make(eventstore.History, 0)
	_, err := p.db.
		WithContext(ctx).
		WithParam("tableName", p.tableName).
		WithParam("aggregateID", aggregateID).
		WithParam("tenantID", tenantID).
		WithParam("fromVersion", fromVersion).
		WithParam("toVersion", toVersion).
		Query(&history, selectRecordsSQL)
	if err != nil && err != pg.ErrNoRows {
		return nil, errors.E(op, errors.Internal, err)
	}

	return history, nil
}

func (p *PgStore) checkIdempotent(ctx context.Context, aggregateID model.ID, tenantID model.ID, records []*eventstore.Record) error {
	const op errors.Op = "pgstore/PgStore.checkIdempotent"

	segments := eventstore.History(records)
	sort.Sort(segments)

	fromVersion := segments[0].Version
	toVersion := segments[len(segments)-1].Version

	persisted, err := p.Load(ctx, aggregateID, tenantID, fromVersion, toVersion)
	if err != nil {
		return err
	}

	if !reflect.DeepEqual(segments, persisted) {
		return errors.E(op, errors.Internal, fmt.Sprintf("conflicting records of aggregate with aggregateID %s detected", aggregateID))
	}

	return nil

}

// Save the provided serialized records to PgStore
func (p *PgStore) Save(ctx context.Context, aggregateID model.ID, tenantID model.ID, records []*eventstore.Record) error {
	const op errors.Op = "pgstore/PgStore.Save"

	if len(records) == 0 {
		return nil
	}

	var maxVersion model.Version
	_, err := p.db.
		WithParam("tableName", p.tableName).
		WithParam("aggregateID", aggregateID).
		Query(&maxVersion, selectMaxVersionSQL)
	if err != nil && err != pg.ErrNoRows {
		return errors.E(op, errors.Internal, err)
	}

	items := eventstore.History(records)
	sort.Sort(items)

	if maxVersion >= items[0].Version {
		if err := p.checkIdempotent(ctx, aggregateID, tenantID, records); err != nil {
			return err
		}
	}

	if _, err := p.db.ModelContext(ctx, &records).Insert(); err != nil {
		return err
	}

	return nil
}

// New returns a Postgres backed store
func New(options *pg.Options, logger logrus.FieldLogger) eventstore.Store {
	logger = logger.WithField("component", "PgStore")
	logger.Infof("Postgres Store: connection=postgresql://%s/%s", options.Addr, options.Database)

	db := pg.Connect(options)
	db.AddQueryHook(NewDebugHook(logger))

	return &PgStore{
		db:     db,
		logger: logger,
	}
}

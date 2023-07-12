package pgstore

import (
	"context"
	"time"

	"github.com/go-pg/pg/v10"
	"github.com/sirupsen/logrus"
)

type DebugHook struct {
	logger logrus.FieldLogger
}

func NewDebugHook(logger logrus.FieldLogger) *DebugHook {
	return &DebugHook{logger: logger}
}

var _ pg.QueryHook = (*DebugHook)(nil)

func (d DebugHook) BeforeQuery(ctx context.Context, event *pg.QueryEvent) (context.Context, error) {
	return ctx, nil
}

func (d DebugHook) AfterQuery(ctx context.Context, event *pg.QueryEvent) error {
	d.logger.WithField("latency", time.Since(event.StartTime))

	query, err := event.FormattedQuery()
	if err != nil {
		d.logger.Errorf("failed to format query: %v", err)
		return err
	}

	if event.Err != nil {
		d.logger.Errorf("error %s executing query: %s", event.Err, query)
	} else {
		d.logger.Debugf("Query processed: %s", query)
	}

	return nil
}

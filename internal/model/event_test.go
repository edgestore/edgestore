package model

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestModel(t *testing.T) {
	now := time.Now()
	m := EventModel{
		ID:      ID("model_1"),
		Version: 123,
		At:      &now,
	}

	assert.EqualValues(t, m.EventID(), "model_1")
	assert.EqualValues(t, m.EventVersion(), 123)
	assert.EqualValues(t, m.EventAt().Unix(), m.At.Unix())
}

type Custom struct {
	EventModel
}

func (c Custom) EventType() string {
	return "custom"
}

func TestEventType(t *testing.T) {
	m := Custom{}
	eventType, _ := EventType(m)
	assert.Equal(t, "custom", eventType)
}

package eventstore

import (
	"testing"

	"github.com/edgestore/edgestore/internal/model"
	"github.com/stretchr/testify/assert"
)

type EntitySetName struct {
	model.EventModel
	Name string
}

func TestNewJSONSerializer(t *testing.T) {
	event := EntitySetName{
		EventModel: model.EventModel{
			ID:      "entity_foo",
			Version: 123,
		},
		Name: "foo",
	}

	serializer := NewJSONSerializer(event)
	record, err := serializer.MarshalEvent(event)
	assert.Nil(t, err)
	assert.NotNil(t, record)

	v, err := serializer.UnmarshalEvent(record)
	assert.Nil(t, err)

	found, ok := v.(*EntitySetName)
	assert.True(t, ok)
	assert.Equal(t, &event, found)
}

func TestJSONSerializer_MarshalAll(t *testing.T) {
	event := EntitySetName{
		EventModel: model.EventModel{
			ID:      "entity_foo",
			Version: 123,
		},
		Name: "foo",
	}

	serializer := NewJSONSerializer(event)
	history, err := serializer.MarshalAll(event)
	assert.Nil(t, err)
	assert.NotNil(t, history)

	v, err := serializer.UnmarshalEvent(history[0])
	assert.Nil(t, err)

	found, ok := v.(*EntitySetName)
	assert.True(t, ok)
	assert.Equal(t, &event, found)
}

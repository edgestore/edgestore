package eventstore

import (
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

func TestNewMemStore(t *testing.T) {
	store := NewInMemory(logrus.New())
	assert.NotNil(t, store)
}

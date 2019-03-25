package model

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewPagination(t *testing.T) {
	p := NewPagination(20, 5)
	assert.Equal(t, 20, p.Limit)
	assert.Equal(t, 100, p.Offset)
}

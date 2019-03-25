package stats

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestGetStats(t *testing.T) {
	s1 := GetStats("vtest")
	assert.NotNil(t, s1)

	s2 := GetStats("vtest")
	assert.NotNil(t, s2)
	assert.True(t, s2.Time > s1.Time)
}

package eventstore

import (
	"sort"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestHistory_Swap(t *testing.T) {
	h := History{
		{Version: 3},
		{Version: 1},
		{Version: 2},
	}

	sort.Sort(h)
	assert.EqualValues(t, 1, h[0].Version)
	assert.EqualValues(t, 2, h[1].Version)
	assert.EqualValues(t, 3, h[2].Version)
}

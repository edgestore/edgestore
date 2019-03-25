package model

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

var testData = []struct {
	id    string
	valid bool
}{
	{"9be47034-289a-11e9-b210-d663bd873d93", true},
	{"1eb4037f-979e-4a5e-9468-da84cd0f3cf5", true},
	{"9be47034-289a-11e9-b210-d663bd873d9", false},
	{"tag", false},
}

func TestIsValidUUID(t *testing.T) {
	for _, data := range testData {
		assert.Equal(t, data.valid, IsValidUUID(data.id), "%s valid: %t", data.id, data.valid)
	}
}

var v4TestData = []struct {
	id    string
	valid bool
}{
	{"1eb4037f-979e-4a5e-9468-da84cd0f3cf5", true},
	{"9be47034-289a-11e9-b210-d663bd873d93", false},
	{"1eb4037f-979e-4a5e-9468-da84cd0f3cf", false},
	{"1eb4037f-979e-0a5e-9468-da84cd0f3cf5", false},
	{"tag", false},
}

func TestIsValidUUIDV4(t *testing.T) {
	for _, data := range v4TestData {
		assert.Equal(t, data.valid, IsValidUUIDV4(data.id), "%s valid: %t", data.id, data.valid)
	}
}

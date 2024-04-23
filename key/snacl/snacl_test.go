package snacl

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestConstantTimeCompare(t *testing.T) {
	a := []byte("Hello World")
	b := []byte("Hello World")

	ret := constantTimeCompare(a, b)
	assert.Equal(t, 1, ret)
}

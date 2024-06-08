package tests

import (
	"testing"

	"bisonai.com/orakl/node/pkg/utils/set"
	"github.com/stretchr/testify/assert"
)

func TestSetBasicOperations(t *testing.T) {
	s := set.NewSet[int]()
	assert.Equal(t, 0, s.Size())

	s.Add(1)
	assert.Equal(t, 1, s.Size())
	assert.True(t, s.Contains(1))
	assert.False(t, s.Contains(2))

	s.Remove(1)
	assert.Equal(t, 0, s.Size())
	assert.False(t, s.Contains(1))
}

package tests

import (
	"testing"

	"bisonai.com/miko/node/pkg/utils/set"
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

func TestSetAddDuplicates(t *testing.T) {
	s := set.NewSet[int]()
	s.Add(1)
	s.Add(1) // Attempt to add duplicate
	assert.Equal(t, 1, s.Size(), "Set should not allow duplicates")
	assert.True(t, s.Contains(1))
}

func TestSetOperationsOnEmpty(t *testing.T) {
	s := set.NewSet[int]()
	assert.False(t, s.Contains(1), "Empty set should not contain any element")
	s.Remove(1) // Attempt to remove from empty set
	assert.Equal(t, 0, s.Size(), "Size should remain 0 after remove operation on empty set")
}

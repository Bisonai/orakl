//nolint:all
package event

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLoadExpectedEventIntervals(t *testing.T) {
	configs, err := loadExpectedEventIntervals()
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	fmt.Println(configs)
	assert.Greater(t, len(configs), 0)
}

package tests

import (
	"context"
	"testing"

	"bisonai.com/orakl/node/pkg/db"
)

func TestPGSGetPoolSingleton(t *testing.T) {
	ctx := context.Background()

	// Call GetPool multiple times
	pool1, err := db.GetPool(ctx)
	if err != nil {
		t.Fatalf("GetPool failed: %v", err)
	}
	defer db.ClosePool()

	pool2, err := db.GetPool(ctx)
	if err != nil {
		t.Fatalf("GetPool failed: %v", err)
	}

	// Check that the returned instances are the same
	if pool1 != pool2 {
		t.Errorf("GetPool did not return the same instance")
	}

}

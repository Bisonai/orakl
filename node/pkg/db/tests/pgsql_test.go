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

	pool, err := db.GetPool(ctx)

	// Create a temporary table
	_, err = pool.Exec(ctx, `CREATE TEMPORARY TABLE test (id SERIAL PRIMARY KEY, name TEXT)`)
	if err != nil {
		t.Fatalf("Failed to create temporary table: %v", err)
	}

	// Insert some test data
	_, err = pool.Exec(ctx, `INSERT INTO test (name) VALUES ('Alice'), ('Bob')`)
	if err != nil {
		t.Fatalf("Failed to insert test data: %v", err)
	}

	// Run a query using your helper function
	rows, err := db.Query(ctx, `SELECT * FROM test WHERE name = @name`, map[string]any{"name": "Alice"})
	if err != nil {
		t.Fatalf("Query failed: %v", err)
	}

	// Check the results
	var id int
	var name string
	for rows.Next() {
		err = rows.Scan(&id, &name)
		if err != nil {
			t.Fatalf("Failed to scan row: %v", err)
		}

		if id != 1 || name != "Alice" {
			t.Errorf("Unexpected row: got %d %s, want 1 Alice", id, name)
		}
	}

	// Check for any error that occurred while iterating over the rows
	if rows.Err() != nil {
		t.Fatalf("Rows iteration failed: %v", rows.Err())
	}
}

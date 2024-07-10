package test

import (
	"context"
	"testing"

	"bisonai.com/orakl/node/pkg/dal/utils/multipgs"
)

func TestNewDatabase(t *testing.T) {
	ctx := context.Background()

	// Test with a valid connection string
	_, err := multipgs.NewDatabase(ctx, "DATABASE_URL")
	if err != nil {
		t.Errorf("NewDatabase with valid connection failed: %v", err)
	}

	// Test with an invalid connection string
	_, err = multipgs.NewDatabase(ctx, "INVALID")
	if err == nil {
		t.Error("NewDatabase with invalid connection did not return error")
	}
}

func TestQueryWithoutResult(t *testing.T) {
	ctx := context.Background()

	pool, err := multipgs.NewDatabase(ctx, "DATABASE_URL")
	if err != nil {
		t.Errorf("NewDatabase failed: %v", err)
	}

	// Create a temporary table
	_, err = pool.Exec(ctx, `CREATE TEMPORARY TABLE test (id SERIAL PRIMARY KEY, name TEXT)`)
	if err != nil {
		t.Fatalf("Failed to create temporary table: %v", err)
	}
	defer func() {
		// Clean up the temporary table
		_, err = pool.Exec(ctx, "DROP TABLE test")
		if err != nil {
			t.Fatalf("Failed to drop table: %v", err)
		}
	}()

	// Insert some test data
	_, err = pool.Exec(ctx, `INSERT INTO test (name) VALUES ('Alice'), ('Bob')`)
	if err != nil {
		t.Fatalf("Failed to insert test data: %v", err)
	}

	// Test with a valid query
	err = multipgs.QueryWithoutResult(ctx, "DATABASE_URL", "SELECT FROM test WHERE id = @name", map[string]interface{}{"name": "Alice"})
	if err != nil {
		t.Errorf("QueryWithoutResult with valid query failed: %v", err)
	}

	// Test with an invalid connection
	err = multipgs.QueryWithoutResult(ctx, "INVALID", "SELECT FROM test WHERE id = @name", map[string]interface{}{"name": "Alice"})
	if err == nil {
		t.Error("QueryWithoutResult with invalid connection did not return error")
	}
}

func TestQueryRow(t *testing.T) {
	ctx := context.Background()

	pool, err := multipgs.NewDatabase(ctx, "DATABASE_URL")
	if err != nil {
		t.Errorf("NewDatabase failed: %v", err)
	}

	// Create a temporary table
	_, err = pool.Exec(ctx, `CREATE TEMPORARY TABLE test (id SERIAL PRIMARY KEY, name TEXT)`)
	if err != nil {
		t.Fatalf("Failed to create temporary table: %v", err)
	}
	defer func() {
		// Clean up the temporary table
		_, err = pool.Exec(ctx, "DROP TABLE test")
		if err != nil {
			t.Fatalf("Failed to drop table: %v", err)
		}
	}()

	// Insert some test data
	_, err = pool.Exec(ctx, `INSERT INTO test (name) VALUES ('Alice'), ('Bob')`)
	if err != nil {
		t.Fatalf("Failed to insert test data: %v", err)
	}

	// Call the function being tested
	result, err := multipgs.QueryRow[struct {
		Name string `db:"name"`
	}](ctx, "DATABASE_URL", `SELECT name FROM test WHERE id = 1`, nil)
	if err != nil {
		t.Fatalf("QueryRow failed: %v", err)
	}

	// Check the result
	if result.Name != "Alice" {
		t.Errorf("Unexpected result: got %s, want Alice", result)
	}

}

func TestQueryRows(t *testing.T) {
	ctx := context.Background()

	pool, err := multipgs.NewDatabase(ctx, "DATABASE_URL")
	if err != nil {
		t.Errorf("NewDatabase failed: %v", err)
	}

	// Create a temporary table
	_, err = pool.Exec(ctx, `CREATE TEMPORARY TABLE test (id SERIAL PRIMARY KEY, name TEXT)`)
	if err != nil {
		t.Fatalf("Failed to create temporary table: %v", err)
	}
	defer func() {
		// Clean up the temporary table
		_, err = pool.Exec(ctx, "DROP TABLE test")
		if err != nil {
			t.Fatalf("Failed to drop table: %v", err)
		}
	}()

	// Insert some test data
	_, err = pool.Exec(ctx, `INSERT INTO test (name) VALUES ('Alice'), ('Bob')`)
	if err != nil {
		t.Fatalf("Failed to insert test data: %v", err)
	}

	// Call the function being tested
	results, err := multipgs.QueryRows[struct {
		ID   int    `db:"id"`
		Name string `db:"name"`
	}](ctx, "DATABASE_URL", `SELECT * FROM test`, nil)
	if err != nil {
		t.Fatalf("QueryRows failed: %v", err)
	}

	// Check the results
	if len(results) != 2 {
		t.Errorf("Unexpected number of results: got %d, want 2", len(results))
	}
}

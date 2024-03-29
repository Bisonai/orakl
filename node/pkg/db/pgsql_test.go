package db

import (
	"context"
	"testing"
)

func TestPGSGetPoolSingleton(t *testing.T) {
	ctx := context.Background()

	// Call GetPool multiple times
	pool1, err := GetPool(ctx)
	if err != nil {
		t.Fatalf("GetPool failed: %v", err)
	}

	pool2, err := GetPool(ctx)
	if err != nil {
		t.Fatalf("GetPool failed: %v", err)
	}

	// Check that the returned instances are the same
	if pool1 != pool2 {
		t.Errorf("GetPool did not return the same instance")
	}
}

func TestPGSGetSet(t *testing.T) {
	ctx := context.Background()
	pool, err := GetPool(ctx)
	if err != nil {
		t.Fatalf("GetPool failed: %v", err)
	}

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
	rows, err := Query(ctx, `SELECT * FROM test WHERE name = @name`, map[string]any{"name": "Alice"})
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

	// Clean up the temporary table
	_, err = pool.Exec(ctx, "DROP TABLE test")
	if err != nil {
		t.Fatalf("Failed to drop table: %v", err)
	}
}
func TestQueryWithoutResult(t *testing.T) {
	ctx := context.Background()
	pool, err := GetPool(ctx)
	if err != nil {
		t.Fatalf("GetPool failed: %v", err)
	}

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

	// Call the function being tested
	err = QueryWithoutResult(ctx, `DELETE FROM test WHERE name = @name`, map[string]interface{}{"name": "Alice"})
	if err != nil {
		t.Fatalf("QueryWithoutResult failed: %v", err)
	}

	// Check if the row was deleted
	rows, err := pool.Query(ctx, `SELECT * FROM test WHERE name = 'Alice'`)
	if err != nil {
		t.Fatalf("Failed to query test data: %v", err)
	}
	defer rows.Close()

	if rows.Next() {
		t.Errorf("Unexpected row found after deletion")
	}

	// Clean up the temporary table
	_, err = pool.Exec(ctx, "DROP TABLE test")
	if err != nil {
		t.Fatalf("Failed to drop table: %v", err)
	}
}
func TestQuery(t *testing.T) {
	ctx := context.Background()
	pool, err := GetPool(ctx)
	if err != nil {
		t.Fatalf("GetPool failed: %v", err)
	}

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

	// Call the function being tested
	rows, err := Query(ctx, `SELECT * FROM test WHERE name = @name`, map[string]interface{}{"name": "Alice"})
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

	// Clean up the temporary table
	_, err = pool.Exec(ctx, "DROP TABLE test")
	if err != nil {
		t.Fatalf("Failed to drop table: %v", err)
	}
}
func TestQueryRow(t *testing.T) {
	ctx := context.Background()
	pool, err := GetPool(ctx)
	if err != nil {
		t.Fatalf("GetPool failed: %v", err)
	}

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

	// Call the function being tested
	result, err := QueryRow[struct {
		Name string `db:"name"`
	}](ctx, `SELECT name FROM test WHERE id = 1`, nil)
	if err != nil {
		t.Fatalf("QueryRow failed: %v", err)
	}

	// Check the result
	if result.Name != "Alice" {
		t.Errorf("Unexpected result: got %s, want Alice", result)
	}

	// Clean up the temporary table
	_, err = pool.Exec(ctx, "DROP TABLE test")
	if err != nil {
		t.Fatalf("Failed to drop table: %v", err)
	}
}
func TestQueryRows(t *testing.T) {
	ctx := context.Background()
	pool, err := GetPool(ctx)
	if err != nil {
		t.Fatalf("GetPool failed: %v", err)
	}

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

	// Call the function being tested
	results, err := QueryRows[struct {
		ID   int    `db:"id"`
		Name string `db:"name"`
	}](ctx, `SELECT * FROM test`, nil)
	if err != nil {
		t.Fatalf("QueryRows failed: %v", err)
	}

	// Check the results
	if len(results) != 2 {
		t.Errorf("Unexpected number of results: got %d, want 2", len(results))
	}

	// Clean up the temporary table
	_, err = pool.Exec(ctx, "DROP TABLE test")
	if err != nil {
		t.Fatalf("Failed to drop table: %v", err)
	}
}

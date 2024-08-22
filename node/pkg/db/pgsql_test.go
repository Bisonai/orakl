package db

import (
	"context"
	"reflect"
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

}

func TestBulkCopy(t *testing.T) {
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
	defer func() {
		_, err = pool.Exec(ctx, "DROP TABLE test")
		if err != nil {
			t.Fatalf("Failed to drop table: %v", err)
		}
	}()

	cnt, err := BulkCopy(ctx, "test", []string{"name"}, [][]any{{"Alice"}, {"Bob"}})
	if err != nil {
		t.Fatalf("BulkInsert failed: %v", err)
	}

	if cnt != 2 {
		t.Errorf("Unexpected number of rows inserted: got %d, want 2", cnt)
	}

}

func TestBulkInsert(t *testing.T) {
	ctx := context.Background()
	pool, err := GetPool(ctx)
	if err != nil {
		t.Fatalf("GetPool failed: %v", err)
	}

	// Create a temporary table
	_, err = pool.Exec(ctx, `CREATE TEMPORARY TABLE test2 (id SERIAL PRIMARY KEY, name TEXT)`)
	if err != nil {
		t.Fatalf("Failed to create temporary table: %v", err)
	}
	defer func() {
		// Clean up the temporary table
		_, err = pool.Exec(ctx, "DROP TABLE test2")
		if err != nil {
			t.Fatalf("Failed to drop table: %v", err)
		}
	}()

	err = BulkInsert(ctx, "test2", []string{"name"}, [][]any{{"Alice"}, {"Bob"}})
	if err != nil {
		t.Fatalf("BulkInsert failed: %v", err)
	}

	// Check the results
	rows, err := pool.Query(ctx, `SELECT * FROM test2`)
	if err != nil {
		t.Fatalf("Failed to query test data: %v", err)
	}
	defer rows.Close()

	var id int
	var name string
	for i := 0; rows.Next(); i++ {
		err = rows.Scan(&id, &name)
		if err != nil {
			t.Fatalf("Failed to scan row: %v", err)
		}

		if name != "Alice" && name != "Bob" {
			t.Errorf("Unexpected row: got %d %s, want 1 Alice or Bob", id, name)
		}
	}

	// Check for any error that occurred while iterating over the rows
	if rows.Err() != nil {
		t.Fatalf("Rows iteration failed: %v", rows.Err())
	}

}

func TestBulkUpsert(t *testing.T) {
	ctx := context.Background()
	pool, err := GetPool(ctx)
	if err != nil {
		t.Fatalf("GetPool failed: %v", err)
	}

	// Create a temporary table
	_, err = pool.Exec(ctx, `CREATE TEMPORARY TABLE test3 (name TEXT PRIMARY KEY, age INT)`)
	if err != nil {
		t.Fatalf("Failed to create temporary table: %v", err)
	}
	defer func() {
		// Clean up the temporary table
		_, err = pool.Exec(ctx, "DROP TABLE test3")
		if err != nil {
			t.Fatalf("Failed to drop table: %v", err)
		}
	}()

	// Insert initial data
	err = BulkInsert(ctx, "test3", []string{"name", "age"}, [][]any{{"Alice", 25}, {"Bob", 30}})
	if err != nil {
		t.Fatalf("BulkInsert failed: %v", err)
	}

	// Update existing data
	err = BulkUpsert(ctx, "test3", []string{"name", "age"}, [][]any{{"Alice", 26}, {"Bob", 31}}, []string{"name"}, []string{"age"})
	if err != nil {
		t.Fatalf("BulkUpsert failed: %v", err)
	}

	// Check the updated results
	rows, err := pool.Query(ctx, `SELECT * FROM test3`)
	if err != nil {
		t.Fatalf("Failed to query test data: %v", err)
	}
	defer rows.Close()

	var name string
	var age int
	for i := 0; rows.Next(); i++ {
		err = rows.Scan(&name, &age)
		if err != nil {
			t.Fatalf("Failed to scan row: %v", err)
		}

		if name == "Alice" && age != 26 {
			t.Errorf("Unexpected row: got %s %d, want Alice 26", name, age)
		}

		if name == "Bob" && age != 31 {
			t.Errorf("Unexpected row: got %s %d, want Bob 31", name, age)
		}
	}

	// Check for any error that occurred while iterating over the rows
	if rows.Err() != nil {
		t.Fatalf("Rows iteration failed: %v", rows.Err())
	}

}
func TestBulkUpdate(t *testing.T) {
	ctx := context.Background()
	pool, err := GetPool(ctx)
	if err != nil {
		t.Fatalf("GetPool failed: %v", err)
	}

	// Create a temporary table
	_, err = pool.Exec(ctx, `CREATE TEMPORARY TABLE test4 (name TEXT PRIMARY KEY, age INT)`)
	if err != nil {
		t.Fatalf("Failed to create temporary table: %v", err)
	}
	defer func() {
		// Clean up the temporary table
		_, err = pool.Exec(ctx, "DROP TABLE test4")
		if err != nil {
			t.Fatalf("Failed to drop table: %v", err)
		}
	}()

	// Insert initial data
	err = BulkInsert(ctx, "test4", []string{"name", "age"}, [][]interface{}{{"Alice", 25}, {"Bob", 30}})
	if err != nil {
		t.Fatalf("BulkInsert failed: %v", err)
	}

	// Update existing data
	err = BulkUpdate(ctx, "test4", []string{"age"}, [][]interface{}{{26, "Alice"}, {31, "Bob"}}, []string{"name"})
	if err != nil {
		t.Fatalf("BulkUpdate failed: %v", err)
	}

	// Check the updated results
	rows, err := pool.Query(ctx, `SELECT * FROM test4`)
	if err != nil {
		t.Fatalf("Failed to query test data: %v", err)
	}
	defer rows.Close()

	var name string
	var age int
	for i := 0; rows.Next(); i++ {
		err = rows.Scan(&name, &age)
		if err != nil {
			t.Fatalf("Failed to scan row: %v", err)
		}

		if name == "Alice" && age != 26 {
			t.Errorf("Unexpected row: got %s %d, want Alice 26", name, age)
		}

		if name == "Bob" && age != 31 {
			t.Errorf("Unexpected row: got %s %d, want Bob 31", name, age)
		}
	}

	// Check for any error that occurred while iterating over the rows
	if rows.Err() != nil {
		t.Fatalf("Rows iteration failed: %v", rows.Err())
	}

}
func TestBulkSelect(t *testing.T) {
	ctx := context.Background()
	pool, err := GetPool(ctx)
	if err != nil {
		t.Fatalf("GetPool failed: %v", err)
	}

	// Create a temporary table
	_, err = pool.Exec(ctx, `CREATE TEMPORARY TABLE test (id SERIAL PRIMARY KEY, name TEXT, age INT)`)
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
	_, err = pool.Exec(ctx, `INSERT INTO test (name, age) VALUES ('Alice', 25), ('Bob', 30)`)
	if err != nil {
		t.Fatalf("Failed to insert test data: %v", err)
	}

	expectedResult := struct {
		Name string `db:"name"`
		Age  int    `db:"age"`
	}{
		Name: "Alice",
		Age:  25,
	}

	expectedResult2 := []struct {
		Name string `db:"name"`
		Age  int    `db:"age"`
	}{
		{
			Name: "Alice",
			Age:  25,
		},
		{
			Name: "Bob",
			Age:  30,
		},
	}

	t.Run("test invalid values", func(t *testing.T) {
		// Call with invalid values
		_, err = BulkSelect[struct {
			Name string `db:"name"`
			Age  int    `db:"age"`
		}](ctx, "", []string{}, []string{}, []any{"Alice", "Bob"})
		if err == nil {
			t.Fatalf("BulkSelect did not return an error")
		}
		_, err = BulkSelect[struct {
			Name string `db:"name"`
			Age  int    `db:"age"`
		}](ctx, "test", []string{}, []string{"name"}, []any{"Alice", "Bob"})
		if err == nil {
			t.Fatalf("BulkSelect did not return an error")
		}
		_, err = BulkSelect[struct {
			Name string `db:"name"`
			Age  int    `db:"age"`
		}](ctx, "test", []string{"name", "age"}, []string{}, []any{"Alice", "Bob"})
		if err == nil {
			t.Fatalf("BulkSelect did not return an error")
		}
		_, err = BulkSelect[struct {
			Name string `db:"name"`
			Age  int    `db:"age"`
		}](ctx, "test", []string{}, []string{"name", "age"}, []any{"Alice", "Bob"})
		if err == nil {
			t.Fatalf("BulkSelect did not return an error")
		}
	})

	t.Run("test single where value", func(t *testing.T) {
		// Call the function being tested
		results, err := BulkSelect[struct {
			Name string `db:"name"`
			Age  int    `db:"age"`
		}](ctx, "test", []string{"name", "age"}, []string{"name"}, []any{"Alice"})
		if err != nil {
			t.Fatalf("BulkSelect failed: %v", err)
		}

		// Check the results
		if len(results) != 1 {
			t.Errorf("Unexpected number of results: got %d, want 1", len(results))
		}

		if !reflect.DeepEqual(results[0], expectedResult) {
			t.Errorf("Unexpected result: got %+v, want %+v", results[0], expectedResult)
		}
	})

	t.Run("test multiple where values", func(t *testing.T) {
		// Call the function with multiple where values
		results, err := BulkSelect[struct {
			Name string `db:"name"`
			Age  int    `db:"age"`
		}](ctx, "test", []string{"name", "age"}, []string{"name"}, []any{"Alice", "Bob"})
		if err != nil {
			t.Fatalf("BulkSelect failed: %v", err)
		}

		// Check the results
		if len(results) != 2 {
			t.Errorf("Unexpected number of results: got %d, want 2", len(results))
		}

		if !reflect.DeepEqual(results, expectedResult2) {
			t.Errorf("Unexpected result: got %+v, want %+v", results, expectedResult2)
		}
	})

	t.Run("test multiple where columns", func(t *testing.T) {
		// Call the function with multiple where columns
		results, err := BulkSelect[struct {
			Name string `db:"name"`
			Age  int    `db:"age"`
		}](ctx, "test", []string{"name", "age"}, []string{"name", "age"}, []any{"Alice", "Bob"}, []any{25, 30})
		if err != nil {
			t.Fatalf("BulkSelect failed: %v", err)
		}

		// Check the results
		if len(results) != 2 {
			t.Errorf("Unexpected number of results: got %d, want 2", len(results))
		}

		if !reflect.DeepEqual(results, expectedResult2) {
			t.Errorf("Unexpected result: got %+v, want %+v", results, expectedResult2)
		}
	})

}

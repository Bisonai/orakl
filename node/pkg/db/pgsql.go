package db

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"sync"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// singleton pattern
// make sure env is loaded from main before calling this

var (
	initPgxOnce sync.Once
	pool        *pgxpool.Pool
	poolErr     error
)

func GetPool(ctx context.Context) (*pgxpool.Pool, error) {
	return getPool(ctx, &initPgxOnce)
}

func getPool(ctx context.Context, once *sync.Once) (*pgxpool.Pool, error) {
	once.Do(func() {
		connectionString := loadPgsqlConnectionString()
		if connectionString == "" {
			poolErr = errors.New("DATABASE_URL is not set")
			return
		}
		pool, poolErr = connectToPgsql(ctx, connectionString)
	})

	return pool, poolErr
}

func connectToPgsql(ctx context.Context, connectionString string) (*pgxpool.Pool, error) {
	return pgxpool.New(ctx, connectionString)
}

func loadPgsqlConnectionString() string {
	return os.Getenv("DATABASE_URL")
}

func QueryWithoutResult(ctx context.Context, queryString string, args map[string]any) error {
	currentPool, err := GetPool(ctx)
	if err != nil {
		return err
	}

	rows, err := query(currentPool, queryString, args)
	if err != nil {
		return err
	}
	rows.Close()
	return nil
}

// Using this raw function is highly unrecommended since rows should be manually closed
func Query(ctx context.Context, queryString string, args map[string]any) (pgx.Rows, error) {
	currentPool, err := GetPool(ctx)
	if err != nil {
		return nil, err
	}

	return query(currentPool, queryString, args)
}

func QueryRow[T any](ctx context.Context, queryString string, args map[string]any) (T, error) {
	var t T
	currentPool, err := GetPool(ctx)
	if err != nil {
		return t, err
	}

	return queryRow[T](currentPool, queryString, args)
}

func QueryRows[T any](ctx context.Context, queryString string, args map[string]any) ([]T, error) {
	currentPool, err := GetPool(ctx)
	if err != nil {
		return nil, err
	}

	return queryRows[T](currentPool, queryString, args)
}

func query(pool *pgxpool.Pool, query string, args map[string]any) (pgx.Rows, error) {
	return pool.Query(context.Background(), query, pgx.NamedArgs(args))
}

func queryRow[T any](pool *pgxpool.Pool, queryString string, args map[string]any) (T, error) {
	var result T
	rows, err := query(pool, queryString, args)
	if err != nil {
		return result, err
	}

	result, err = pgx.CollectOneRow(rows, pgx.RowToStructByName[T])
	if errors.Is(err, pgx.ErrNoRows) {
		return result, nil
	}
	defer rows.Close()
	return result, err
}

func queryRows[T any](pool *pgxpool.Pool, queryString string, args map[string]any) ([]T, error) {
	results := []T{}

	rows, err := query(pool, queryString, args)
	if err != nil {
		return results, err
	}

	results, err = pgx.CollectRows(rows, pgx.RowToStructByName[T])
	if errors.Is(err, pgx.ErrNoRows) {
		return results, nil
	}
	defer rows.Close()
	return results, err
}

func BulkInsert(ctx context.Context, tableName string, columnNames []string, rows [][]any) error {
	currentPool, err := GetPool(ctx)
	if err != nil {
		return err
	}

	var b strings.Builder
	fmt.Fprintf(&b, "INSERT INTO %s (%s) VALUES ", tableName, strings.Join(columnNames, ", "))

	for i, row := range rows {
		if i > 0 {
			b.WriteString(", ")
		}
		b.WriteString("(")
		for j := range row {
			if j > 0 {
				b.WriteString(", ")
			}
			fmt.Fprintf(&b, "$%d", i*len(row)+j+1)
		}
		b.WriteString(")")
	}

	values := make([]any, 0, len(rows)*len(columnNames))
	for _, row := range rows {
		values = append(values, row...)
	}

	_, err = currentPool.Exec(ctx, b.String(), values...)
	if err != nil {
		return err
	}
	return nil
}

func BulkUpsert(ctx context.Context, tableName string, columnNames []string, rows [][]any, conflictColumns []string, updateColumns []string) error {
	currentPool, err := GetPool(ctx)
	if err != nil {
		return err
	}

	var b strings.Builder
	fmt.Fprintf(&b, "INSERT INTO %s (%s) VALUES ", tableName, strings.Join(columnNames, ", "))

	for i, row := range rows {
		if i > 0 {
			b.WriteString(", ")
		}
		b.WriteString("(")
		for j := range row {
			if j > 0 {
				b.WriteString(", ")
			}
			fmt.Fprintf(&b, "$%d", i*len(row)+j+1)
		}
		b.WriteString(")")
	}

	b.WriteString(" ON CONFLICT (")
	b.WriteString(strings.Join(conflictColumns, ", "))
	b.WriteString(") DO UPDATE SET ")

	for i, col := range updateColumns {
		if i > 0 {
			b.WriteString(", ")
		}
		fmt.Fprintf(&b, "%s = EXCLUDED.%s", col, col)
	}

	values := make([]any, 0, len(rows)*len(columnNames))
	for _, row := range rows {
		values = append(values, row...)
	}

	_, err = currentPool.Exec(ctx, b.String(), values...)
	if err != nil {
		return err
	}
	return nil
}

// use for large entries of data more than 100+ rows
func BulkCopy(ctx context.Context, tableName string, columnNames []string, rows [][]any) (int64, error) {
	currentPool, err := GetPool(ctx)
	if err != nil {
		return 0, err
	}

	return currentPool.CopyFrom(
		ctx,
		pgx.Identifier{tableName},
		columnNames,
		pgx.CopyFromRows(rows),
	)
}

func ClosePool() {
	if pool != nil {
		pool.Close()
		pool = nil
	}
}

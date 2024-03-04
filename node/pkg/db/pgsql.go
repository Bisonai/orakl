package db

import (
	"context"
	"errors"
	"os"
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

func ClosePool() {
	if pool != nil {
		pool.Close()
		pool = nil
	}
}

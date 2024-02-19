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
)

func GetPool(ctx context.Context) (*pgxpool.Pool, error) {
	return getPool(ctx, &initPgxOnce)
}

func getPool(ctx context.Context, once *sync.Once) (*pgxpool.Pool, error) {
	var err error
	once.Do(func() {
		connectionString := loadPgsqlConnectionString()
		if connectionString == "" {
			err = errors.New("DATABASE_URL is not set")
			return
		}
		pool, err = connectToPgsql(ctx, connectionString)
	})
	return pool, err
}

func connectToPgsql(ctx context.Context, connectionString string) (*pgxpool.Pool, error) {
	return pgxpool.New(ctx, connectionString)
}

func loadPgsqlConnectionString() string {
	return os.Getenv("DATABASE_URL")
}

func Query(ctx context.Context, queryString string, args map[string]any) (pgx.Rows, error) {
	pool, err := GetPool(ctx)
	if err != nil {
		return nil, err
	}
	return query(pool, queryString, args)
}

func QueryRow[T any](ctx context.Context, queryString string, args map[string]any) (T, error) {
	var t T
	pool, err := GetPool(ctx)
	if err != nil {
		return t, err
	}
	return queryRow[T](pool, queryString, args)
}

func QueryRows[T any](ctx context.Context, queryString string, args map[string]any) ([]T, error) {
	pool, err := GetPool(ctx)
	if err != nil {
		return nil, err
	}
	return queryRows[T](pool, queryString, args)
}

func query(pool *pgxpool.Pool, query string, args map[string]any) (pgx.Rows, error) {
	return pool.Query(context.Background(), query, pgx.NamedArgs(args))
}

func queryRow[T any](pool *pgxpool.Pool, _query string, args map[string]any) (T, error) {
	var result T
	rows, err := query(pool, _query, args)
	if err != nil {
		return result, err
	}

	result, err = pgx.CollectOneRow(rows, pgx.RowToStructByName[T])
	if errors.Is(err, pgx.ErrNoRows) {
		return result, nil
	}
	return result, err
}

func queryRows[T any](pool *pgxpool.Pool, _query string, args map[string]any) ([]T, error) {
	results := []T{}

	rows, err := query(pool, _query, args)
	if err != nil {
		return results, err
	}

	results, err = pgx.CollectRows(rows, pgx.RowToStructByName[T])
	if errors.Is(err, pgx.ErrNoRows) {
		return results, nil
	}
	return results, err
}

func ClosePool() {
	if pool != nil {
		pool.Close()
		pool = nil
	}
}

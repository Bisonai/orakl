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
	var err error
	initPgxOnce.Do(func() {
		pool, err = connectPgsql(ctx)
	})

	return pool, err
}

func connectPgsql(ctx context.Context) (*pgxpool.Pool, error) {
	connectionString := os.Getenv("DATABASE_URL")
	if connectionString == "" {
		err := errors.New("DATABASE_URL is not set")
		return nil, err
	}
	return pgxpool.New(ctx, connectionString)
}

func Query(ctx context.Context, queryString string, args map[string]any) (pgx.Rows, error) {
	pool, err := GetPool(ctx)
	if err != nil {
		return nil, err
	}
	return query(ctx, pool, queryString, args)
}

func QueryRow[T any](ctx context.Context, queryString string, args map[string]any) (T, error) {
	var t T
	pool, err := GetPool(ctx)
	if err != nil {
		return t, err
	}
	return queryRow[T](ctx, pool, queryString, args)
}

func QueryRows[T any](ctx context.Context, queryString string, args map[string]any) ([]T, error) {
	pool, err := GetPool(ctx)
	if err != nil {
		return nil, err
	}
	return queryRows[T](ctx, pool, queryString, args)
}

func query(ctx context.Context, pool *pgxpool.Pool, query string, args map[string]any) (pgx.Rows, error) {
	return pool.Query(ctx, query, pgx.NamedArgs(args))
}

func queryRow[T any](ctx context.Context, pool *pgxpool.Pool, _query string, args map[string]any) (T, error) {
	var result T
	rows, err := query(ctx, pool, _query, args)
	if err != nil {
		return result, err
	}

	result, err = pgx.CollectOneRow(rows, pgx.RowToStructByName[T])
	if errors.Is(err, pgx.ErrNoRows) {
		return result, nil
	}
	return result, err
}

func queryRows[T any](ctx context.Context, pool *pgxpool.Pool, _query string, args map[string]any) ([]T, error) {
	results := []T{}

	rows, err := query(ctx, pool, _query, args)
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

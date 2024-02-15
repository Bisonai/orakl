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

func GetPool() (*pgxpool.Pool, error) {
	var err error
	initPgxOnce.Do(func() {
		connectionString := loadPgsqlConnectionString()
		if connectionString == "" {
			err = errors.New("DATABASE_URL is not set")
			return
		}
		pool, err = connectToPgsql(connectionString)
	})
	return pool, err
}

func connectToPgsql(connectionString string) (*pgxpool.Pool, error) {
	return pgxpool.New(context.Background(), connectionString)
}

func loadPgsqlConnectionString() string {
	return os.Getenv("DATABASE_URL")
}

func Query(queryString string, args map[string]any) (pgx.Rows, error) {
	pool, err := GetPool()
	if err != nil {
		return nil, err
	}
	return query(pool, queryString, args)
}

func QueryRow[T any](queryString string, args map[string]any) (T, error) {
	pool, err := GetPool()
	if err != nil {
		return *new(T), err
	}
	return queryRow[T](pool, queryString, args)
}

func QueryRows[T any](queryString string, args map[string]any) ([]T, error) {
	pool, err := GetPool()
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

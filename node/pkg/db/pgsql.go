package db

import (
	"context"
	"errors"
	"os"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PgsqlHelper struct {
	Pool *pgxpool.Pool
}

func NewPgsqlHelper() (*PgsqlHelper, error) {
	connectionString := LoadPgsqlConnectionString()
	pool, err := ConnectToPgsql(connectionString)
	if err != nil {
		return nil, err
	}

	return &PgsqlHelper{Pool: pool}, nil
}

func (p *PgsqlHelper) Query(_query string, args map[string]any) (pgx.Rows, error) {
	return query(p.Pool, _query, args)
}

// go methods doesn't allow type parameters
func (p *PgsqlHelper) QueryRow(query string, args map[string]any) (interface{}, error) {
	return queryRow[interface{}](p.Pool, query, args)
}

func (p *PgsqlHelper) QueryRows(query string, args map[string]any) ([]interface{}, error) {
	return queryRows[interface{}](p.Pool, query, args)

}

func ConnectToPgsql(connectionString string) (*pgxpool.Pool, error) {
	return pgxpool.New(context.Background(), connectionString)
}

func LoadPgsqlConnectionString() string {
	return os.Getenv("DATABASE_URL")
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

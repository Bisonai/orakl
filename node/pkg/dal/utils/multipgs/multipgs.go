package multipgs

import (
	"context"
	"errors"
	"sync"

	errorSentinel "bisonai.com/orakl/node/pkg/error"
	"bisonai.com/orakl/node/pkg/secrets"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog/log"
)

var (
	dbInstances sync.Map
)

func NewDatabase(ctx context.Context, connectionEnv string) (*pgxpool.Pool, error) {
	if db, ok := dbInstances.Load(connectionEnv); ok {
		return db.(*pgxpool.Pool), nil
	}

	connectionString := loadPgsqlConnectionString(connectionEnv)
	if connectionString == "" {
		log.Error().Msg("DATABASE_URL is not set for " + connectionEnv)
		return nil, errorSentinel.ErrDbDatabaseUrlNotFound
	}
	pool, err := connectToPgsql(ctx, connectionString)
	if err != nil {
		return nil, err
	}

	db := pool
	dbInstances.Store(connectionEnv, db)

	return db, nil
}

func CloseAll() {
	dbInstances.Range(func(key, value any) bool {
		value.(*pgxpool.Pool).Close()
		dbInstances.Delete(key)
		return true
	})
}

func loadPgsqlConnectionString(connectionEnv string) string {
	return secrets.GetSecret(connectionEnv)
}

func connectToPgsql(ctx context.Context, connectionString string) (*pgxpool.Pool, error) {
	return pgxpool.New(ctx, connectionString)
}

func query(ctx context.Context, pool *pgxpool.Pool, queryString string, args map[string]any) (pgx.Rows, error) {
	return pool.Query(ctx, queryString, pgx.NamedArgs(args))
}

func QueryWithoutResult(ctx context.Context, dbEnv string, queryString string, args map[string]any) error {
	pool, err := NewDatabase(ctx, dbEnv)
	if err != nil {
		return err
	}

	rows, err := query(ctx, pool, queryString, args)
	if err != nil {
		log.Error().Err(err).Str("query", queryString).Msg("Error querying")
		return err
	}
	defer rows.Close()
	return nil
}

func QueryRow[T any](ctx context.Context, dbEnv string, queryString string, args map[string]any) (T, error) {
	var result T

	pool, err := NewDatabase(ctx, dbEnv)
	if err != nil {
		return result, err
	}

	rows, err := query(ctx, pool, queryString, args)
	if err != nil {
		log.Error().Err(err).Str("query", queryString).Msg("Error querying")
		return result, err
	}

	result, err = pgx.CollectOneRow(rows, pgx.RowToStructByName[T])
	if errors.Is(err, pgx.ErrNoRows) {
		return result, nil
	}
	defer rows.Close()
	return result, err
}

func QueryRows[T any](ctx context.Context, dbEnv string, queryString string, args map[string]any) ([]T, error) {
	results := []T{}

	pool, err := NewDatabase(ctx, dbEnv)
	if err != nil {
		return results, err
	}

	rows, err := query(ctx, pool, queryString, args)
	if err != nil {
		log.Error().Err(err).Str("query", queryString).Msg("Error querying")
		return results, err
	}

	results, err = pgx.CollectRows(rows, pgx.RowToStructByName[T])
	if errors.Is(err, pgx.ErrNoRows) {
		return results, nil
	}
	defer rows.Close()
	return results, err
}

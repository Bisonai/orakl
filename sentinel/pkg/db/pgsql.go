package db

import (
	"context"
	"errors"
	"sync"

	"bisonai.com/orakl/sentinel/pkg/secrets"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog/log"
)

// copy pasted from node/pkg/db/pgsql.go
var (
	initPgxOnce sync.Once
	pool        *pgxpool.Pool
	poolErr     error
)

func GetPool(ctx context.Context) (*pgxpool.Pool, error) {
	log.Debug().Msg("Attempting to connect to PostgreSQL")
	return getPool(ctx, &initPgxOnce)
}

func getPool(ctx context.Context, once *sync.Once) (*pgxpool.Pool, error) {
	once.Do(func() {
		connectionString := loadPgsqlConnectionString()
		if connectionString == "" {
			log.Error().Msg("DATABASE_URL is not set")
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
	return secrets.GetSecret("DATABASE_URL")
}

func Query(ctx context.Context, queryString string, args map[string]any) (pgx.Rows, error) {
	currentPool, err := GetPool(ctx)
	if err != nil {
		log.Error().Err(err).Msg("Error getting pool")
		return nil, err
	}

	return query(currentPool, queryString, args)
}

func QueryRow[T any](ctx context.Context, queryString string, args map[string]any) (T, error) {
	var t T
	currentPool, err := GetPool(ctx)
	if err != nil {
		log.Error().Err(err).Msg("Error getting pool")
		return t, err
	}

	return queryRow[T](currentPool, queryString, args)
}

func QueryRows[T any](ctx context.Context, queryString string, args map[string]any) ([]T, error) {
	currentPool, err := GetPool(ctx)
	if err != nil {
		log.Error().Err(err).Msg("Error getting pool")
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
		log.Error().Err(err).Str("query", queryString).Msg("Error querying")
		return result, err
	}
	log.Debug().Msg("Query executed successfully")

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
		log.Error().Err(err).Str("query", queryString).Msg("Error querying")
		return results, err
	}
	log.Debug().Msg("Query executed successfully")

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

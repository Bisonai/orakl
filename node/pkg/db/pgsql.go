package db

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"

	errorSentinel "bisonai.com/orakl/node/pkg/error"
	"bisonai.com/orakl/node/pkg/secrets"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog/log"
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
			log.Error().Msg("DATABASE_URL is not set")
			poolErr = errorSentinel.ErrDbDatabaseUrlNotFound
			return
		}
		pool, poolErr = connectToPgsql(ctx, connectionString)
		if poolErr != nil {
			pool.Close()
		}
	})

	return pool, poolErr
}

func connectToPgsql(ctx context.Context, connectionString string) (*pgxpool.Pool, error) {
	pool, err := pgxpool.New(ctx, connectionString)
	if err != nil {
		log.Error().Err(err).Msg("failed to create connection pool")
		return nil, err
	}

	if err = pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, err
	}

	return pool, nil
}

func loadPgsqlConnectionString() string {
	return secrets.GetSecret("DATABASE_URL")
}

func QueryWithoutResult(ctx context.Context, queryString string, args map[string]any) error {
	currentPool, err := GetPool(ctx)
	if err != nil {
		log.Error().Err(err).Msg("Error getting pool")
		return err
	}

	rows, err := query(ctx, currentPool, queryString, args)
	if err != nil {
		log.Error().Err(err).Msg("Error querying")
		return err
	}
	rows.Close()
	return nil
}

func QueryRow[T any](ctx context.Context, queryString string, args map[string]any) (T, error) {
	var t T
	currentPool, err := GetPool(ctx)
	if err != nil {
		log.Error().Err(err).Msg("Error getting pool")
		return t, err
	}

	return queryRow[T](ctx, currentPool, queryString, args)
}

func QueryRows[T any](ctx context.Context, queryString string, args map[string]any) ([]T, error) {
	currentPool, err := GetPool(ctx)
	if err != nil {
		log.Error().Err(err).Msg("Error getting pool")
		return nil, err
	}

	return queryRows[T](ctx, currentPool, queryString, args)
}

func query(ctx context.Context, pool *pgxpool.Pool, query string, args map[string]any) (pgx.Rows, error) {
	return pool.Query(ctx, query, pgx.NamedArgs(args))
}

func queryRow[T any](ctx context.Context, pool *pgxpool.Pool, queryString string, args map[string]any) (T, error) {
	var result T
	rows, err := query(ctx, pool, queryString, args)
	if err != nil {
		log.Error().Err(err).Str("query", queryString).Msg("Error querying")
		return result, err
	}
	defer rows.Close()

	result, err = pgx.CollectOneRow(rows, pgx.RowToStructByName[T])
	if errors.Is(err, pgx.ErrNoRows) {
		return result, nil
	}

	return result, err
}

func queryRows[T any](ctx context.Context, pool *pgxpool.Pool, queryString string, args map[string]any) ([]T, error) {
	results := []T{}

	rows, err := query(ctx, pool, queryString, args)
	if err != nil {
		log.Error().Err(err).Str("query", queryString).Msg("Error querying")
		return results, err
	}
	defer rows.Close()

	results, err = pgx.CollectRows(rows, pgx.RowToStructByName[T])
	if errors.Is(err, pgx.ErrNoRows) {
		return results, nil
	}

	return results, err
}

func BulkSelect[T any](ctx context.Context, tableName string, columnNames []string, whereColumns []string, whereValues ...[]interface{}) ([]T, error) {
	results := []T{}
	if tableName == "" {
		log.Error().Msg("tableName must not be empty")
		return results, errorSentinel.ErrDbEmptyTableNameParam
	}
	if len(columnNames) == 0 {
		log.Error().Msg("columnNames must not be empty")
		return results, errorSentinel.ErrDbEmptyColumnNamesParam
	}
	if len(whereColumns) == 0 {
		log.Error().Msg("whereColumns must not be empty")
		return results, errorSentinel.ErrDbEmptyWhereColumnsParam
	}
	if len(whereColumns) != len(whereValues) {
		log.Error().Strs("whereColumns", whereColumns).Any("whereValues", whereValues).Msg("whereColumns and whereValues must have the same length")
		return results, errorSentinel.ErrDbWhereColumnValueLengthMismatch
	}

	currentPool, err := GetPool(ctx)
	if err != nil {
		log.Error().Err(err).Msg("Error getting pool")
		return results, err
	}

	var b strings.Builder
	fmt.Fprintf(&b, "SELECT %s FROM %s WHERE ", strings.Join(columnNames, ", "), tableName)

	argIndex := 1
	for i, col := range whereColumns {
		if i > 0 {
			b.WriteString(" AND ")
		}

		values := whereValues[i]

		placeholders := make([]string, len(values))
		for j := range values {
			placeholders[j] = fmt.Sprintf("$%d", argIndex)
			argIndex++
		}

		fmt.Fprintf(&b, "%s IN (%s)", col, strings.Join(placeholders, ","))
	}

	flatWhereValues := make([]interface{}, 0, argIndex-1)
	for _, v := range whereValues {
		flatWhereValues = append(flatWhereValues, v...)
	}

	rows, err := currentPool.Query(ctx, b.String(), flatWhereValues...)
	if err != nil {
		log.Error().Err(err).Str("query", b.String()).Msg("Error querying")
		return results, err
	}
	defer rows.Close()

	results, err = pgx.CollectRows(rows, pgx.RowToStructByName[T])
	if errors.Is(err, pgx.ErrNoRows) {
		return results, nil
	}

	return results, err
}

func BulkInsert(ctx context.Context, tableName string, columnNames []string, rows [][]any) error {
	currentPool, err := GetPool(ctx)
	if err != nil {
		log.Error().Err(err).Msg("Error getting pool")
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
		log.Error().Err(err).Str("query", b.String()).Msg("Error executing query")
		return err
	}
	return nil
}

func BulkUpsert(ctx context.Context, tableName string, columnNames []string, rows [][]any, conflictColumns []string, updateColumns []string) error {
	currentPool, err := GetPool(ctx)
	if err != nil {
		log.Error().Err(err).Msg("Error getting pool")
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
		log.Error().Err(err).Str("query", b.String()).Msg("Error executing query")
		return err
	}
	return nil
}

func BulkUpdate(ctx context.Context, tableName string, columnNames []string, rows [][]interface{}, whereColumns []string) error {
	currentPool, err := GetPool(ctx)
	if err != nil {
		log.Error().Err(err).Msg("Error getting pool")
		return err
	}

	batch := &pgx.Batch{}

	for _, row := range rows {
		var b strings.Builder
		fmt.Fprintf(&b, "UPDATE %s SET ", tableName)

		for i, col := range columnNames {
			if i > 0 {
				b.WriteString(", ")
			}
			fmt.Fprintf(&b, "%s = $%d", col, i+1)
		}

		b.WriteString(" WHERE ")

		for i, col := range whereColumns {
			if i > 0 {
				b.WriteString(" AND ")
			}
			fmt.Fprintf(&b, "%s = $%d", col, i+len(columnNames)+1)
		}
		batch.Queue(b.String(), row...)
	}

	br := currentPool.SendBatch(ctx, batch)
	defer br.Close()

	for range rows {
		_, err = br.Exec()
		if err != nil {
			log.Error().Err(err).Msg("Error executing batch")
			return err
		}
	}

	return nil
}

// use for large entries of data more than 100+ rows
func BulkCopy(ctx context.Context, tableName string, columnNames []string, rows [][]any) (int64, error) {
	currentPool, err := GetPool(ctx)
	if err != nil {
		log.Error().Err(err).Msg("Error getting pool")
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

// use if multiple different db connection is required
// carefully generate pool, and always close after use
func GetTransientPool(ctx context.Context, connectionString string) (*pgxpool.Pool, error) {
	return connectToPgsql(ctx, connectionString)
}

func QueryRowTransient[T any](ctx context.Context, pool *pgxpool.Pool, queryString string, args map[string]any) (T, error) {
	return queryRow[T](ctx, pool, queryString, args)
}

func QueryRowsTransient[T any](ctx context.Context, pool *pgxpool.Pool, queryString string, args map[string]any) ([]T, error) {
	return queryRows[T](ctx, pool, queryString, args)
}

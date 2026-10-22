// Package keystore persists DAL API keys in a local SQLite file.
package keystore

import (
	"context"
	"database/sql"
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"

	_ "modernc.org/sqlite" // pure-Go SQLite driver, registers "sqlite"
)

// EnvPath is the env var holding the SQLite file path.
const EnvPath = "DAL_SQLITE_PATH"

const schema = `
CREATE TABLE IF NOT EXISTS keys (
	id          INTEGER PRIMARY KEY AUTOINCREMENT,
	key         TEXT NOT NULL UNIQUE,
	description TEXT NOT NULL DEFAULT ''
);`

const upsertQuery = "INSERT INTO keys (key, description) VALUES (?, ?) ON CONFLICT(key) DO UPDATE SET description = excluded.description"

var ErrPathNotSet = errors.New(EnvPath + " environment variable not set")

type Key struct {
	Key         string
	Description string
}

type Store struct {
	db *sql.DB
}

// OpenFromEnv opens the store at the path in DAL_SQLITE_PATH.
func OpenFromEnv(ctx context.Context) (*Store, error) {
	path := os.Getenv(EnvPath)
	if path == "" {
		return nil, ErrPathNotSet
	}
	return Open(ctx, path)
}

// Open opens (creating if missing) the SQLite file at path and ensures the schema.
func Open(ctx context.Context, path string) (*Store, error) {
	q := url.Values{}
	q.Add("_pragma", "busy_timeout(5000)")
	q.Add("_pragma", "journal_mode(WAL)")
	dsn := "file:" + path + "?" + q.Encode()

	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("open sqlite %s: %w", path, err)
	}
	if _, err = db.ExecContext(ctx, schema); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("init sqlite schema %s: %w", path, err)
	}
	return &Store{db: db}, nil
}

func (s *Store) Close() error {
	return s.db.Close()
}

// Lookup returns the description of key and whether it exists.
func (s *Store) Lookup(ctx context.Context, key string) (string, bool, error) {
	var description string
	err := s.db.QueryRowContext(ctx, "SELECT description FROM keys WHERE key = ?", key).Scan(&description)
	if errors.Is(err, sql.ErrNoRows) {
		return "", false, nil
	}
	if err != nil {
		return "", false, err
	}
	return description, true, nil
}

// Upsert adds key, or updates its description if it already exists.
func (s *Store) Upsert(ctx context.Context, key, description string) error {
	if key == "" {
		return errors.New("empty key")
	}
	_, err := s.db.ExecContext(ctx, upsertQuery, key, description)
	return err
}

// Remove deletes key; reports whether a row was removed.
func (s *Store) Remove(ctx context.Context, key string) (bool, error) {
	res, err := s.db.ExecContext(ctx, "DELETE FROM keys WHERE key = ?", key)
	if err != nil {
		return false, err
	}
	n, err := res.RowsAffected()
	return n > 0, err
}

// List returns all keys ordered by id.
func (s *Store) List(ctx context.Context) ([]Key, error) {
	rows, err := s.db.QueryContext(ctx, "SELECT key, description FROM keys ORDER BY id")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []Key
	for rows.Next() {
		var k Key
		if err := rows.Scan(&k.Key, &k.Description); err != nil {
			return nil, err
		}
		result = append(result, k)
	}
	return result, rows.Err()
}

// Count returns the number of stored keys.
func (s *Store) Count(ctx context.Context) (int, error) {
	var n int
	err := s.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM keys").Scan(&n)
	return n, err
}

// ImportCSV upserts `key,description` rows (no header) from r in one
// transaction. Returns the number of rows imported.
func (s *Store) ImportCSV(ctx context.Context, r io.Reader) (int, error) {
	reader := csv.NewReader(r)
	reader.FieldsPerRecord = -1

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, err
	}
	defer func() { _ = tx.Rollback() }()

	n := 0
	for {
		record, err := reader.Read()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return 0, err
		}
		if len(record) == 0 || record[0] == "" {
			return 0, fmt.Errorf("line %d: empty key", n+1)
		}
		description := ""
		if len(record) > 1 {
			description = record[1]
		}
		if _, err := tx.ExecContext(ctx, upsertQuery, record[0], description); err != nil {
			return 0, err
		}
		n++
	}
	return n, tx.Commit()
}

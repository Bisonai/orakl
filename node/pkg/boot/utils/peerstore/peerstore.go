// Package peerstore persists boot API peers in a local SQLite file.
package peerstore

import (
	"bufio"
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"
	"strings"

	_ "modernc.org/sqlite" // pure-Go SQLite driver, registers "sqlite"
)

// EnvPath is the env var holding the SQLite file path.
const EnvPath = "BOOT_SQLITE_PATH"

const schema = `
CREATE TABLE IF NOT EXISTS peers (
	id  INTEGER PRIMARY KEY AUTOINCREMENT,
	url TEXT NOT NULL UNIQUE
);`

const insertQuery = "INSERT INTO peers (url) VALUES (?) ON CONFLICT(url) DO NOTHING RETURNING id, url"

var ErrPathNotSet = errors.New(EnvPath + " environment variable not set")

type Peer struct {
	ID  int64  `db:"id" json:"id"`
	Url string `db:"url" json:"url"`
}

type Store struct {
	db *sql.DB
}

// OpenFromEnv opens the store at the path in BOOT_SQLITE_PATH. With
// mustExist, a missing file is an error instead of a new empty store, so a
// wrong path or unmounted volume fails loudly.
func OpenFromEnv(ctx context.Context, mustExist bool) (*Store, error) {
	path := os.Getenv(EnvPath)
	if path == "" {
		return nil, ErrPathNotSet
	}
	if mustExist {
		if _, err := os.Stat(path); err != nil {
			return nil, fmt.Errorf("peer store %s: %w", path, err)
		}
	}
	return Open(ctx, path)
}

// Open opens (creating if missing) the SQLite file at path and ensures the schema.
func Open(ctx context.Context, path string) (*Store, error) {
	// SQLite decodes '%HH' in file: URIs, so '%' would open a different file.
	if strings.ContainsAny(path, "?#%") {
		return nil, fmt.Errorf("peer store path %q must not contain '?', '#' or '%%'", path)
	}
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

// Ping checks the store is reachable.
func (s *Store) Ping(ctx context.Context) error {
	return s.db.PingContext(ctx)
}

// Insert adds peerURL. If it already exists nothing changes and it returns a
// zero Peer with inserted=false (same as Postgres ON CONFLICT DO NOTHING).
func (s *Store) Insert(ctx context.Context, peerURL string) (Peer, bool, error) {
	if peerURL == "" {
		return Peer{}, false, errors.New("empty url")
	}
	var p Peer
	err := s.db.QueryRowContext(ctx, insertQuery, peerURL).Scan(&p.ID, &p.Url)
	if errors.Is(err, sql.ErrNoRows) {
		return Peer{}, false, nil
	}
	if err != nil {
		return Peer{}, false, err
	}
	return p, true, nil
}

// List returns all peers ordered by id; never nil, so it encodes as [] in JSON.
func (s *Store) List(ctx context.Context) ([]Peer, error) {
	rows, err := s.db.QueryContext(ctx, "SELECT id, url FROM peers ORDER BY id")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := []Peer{}
	for rows.Next() {
		var p Peer
		if err := rows.Scan(&p.ID, &p.Url); err != nil {
			return nil, err
		}
		result = append(result, p)
	}
	return result, rows.Err()
}

// DeleteByID deletes the peer with id; reports whether a row was removed.
func (s *Store) DeleteByID(ctx context.Context, id int64) (bool, error) {
	res, err := s.db.ExecContext(ctx, "DELETE FROM peers WHERE id = ?", id)
	if err != nil {
		return false, err
	}
	n, err := res.RowsAffected()
	return n > 0, err
}

// ImportURLs inserts one url per line from r in one transaction, skipping
// blank lines and urls already stored. Returns the number of new peers.
func (s *Store) ImportURLs(ctx context.Context, r io.Reader) (int, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, err
	}
	defer func() { _ = tx.Rollback() }()

	scanner := bufio.NewScanner(r)
	n := 0
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		res, err := tx.ExecContext(ctx, "INSERT INTO peers (url) VALUES (?) ON CONFLICT(url) DO NOTHING", line)
		if err != nil {
			return 0, err
		}
		added, err := res.RowsAffected()
		if err != nil {
			return 0, err
		}
		n += int(added)
	}
	if err := scanner.Err(); err != nil {
		return 0, err
	}
	return n, tx.Commit()
}

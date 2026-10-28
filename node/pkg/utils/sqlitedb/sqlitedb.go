// Package sqlitedb opens local SQLite state files (pure-Go modernc driver).
package sqlitedb

import (
	"context"
	"database/sql"
	"fmt"
	"net/url"
	"os"
	"strings"

	_ "modernc.org/sqlite" // pure-Go SQLite driver, registers "sqlite"
)

// PathFromEnv returns the SQLite file path in envVar. With mustExist, a
// missing file is an error instead of a new empty store, so a wrong path or
// unmounted volume fails loudly.
func PathFromEnv(envVar string, mustExist bool) (string, error) {
	path := os.Getenv(envVar)
	if path == "" {
		return "", fmt.Errorf("%s environment variable not set", envVar)
	}
	if mustExist {
		if _, err := os.Stat(path); err != nil {
			return "", fmt.Errorf("sqlite file %s: %w", path, err)
		}
	}
	return path, nil
}

// Open opens (creating if missing) the SQLite file at path and applies schema.
func Open(ctx context.Context, path, schema string) (*sql.DB, error) {
	// SQLite decodes '%HH' in file: URIs, so '%' would open a different file.
	if strings.ContainsAny(path, "?#%") {
		return nil, fmt.Errorf("sqlite path %q must not contain '?', '#' or '%%'", path)
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
	return db, nil
}

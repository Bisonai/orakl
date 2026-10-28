// Package nodestate persists nodebin state (signer keyring + latest round per
// config) in a local SQLite file. Not used at runtime yet (issue #2695).
package nodestate

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"bisonai.com/miko/node/pkg/chain/utils"
	"bisonai.com/miko/node/pkg/utils/sqlitedb"
)

// EnvPath is the env var holding the SQLite file path.
const EnvPath = "NODE_SQLITE_PATH"

// signer_key mirrors the Postgres table (migration 000041); pk stays encrypted.
// Timestamps are RFC3339Nano UTC text.
const schema = `
CREATE TABLE IF NOT EXISTS signer_key (
	id         INTEGER PRIMARY KEY AUTOINCREMENT,
	address    TEXT,
	pk         TEXT NOT NULL,
	state      TEXT NOT NULL DEFAULT 'pending' CHECK (state IN ('active', 'pending', 'retired')),
	tx_hash    TEXT,
	created_at TEXT NOT NULL,
	updated_at TEXT NOT NULL
);
CREATE UNIQUE INDEX IF NOT EXISTS signer_key_one_active ON signer_key (state)   WHERE state = 'active';
CREATE UNIQUE INDEX IF NOT EXISTS signer_key_addr       ON signer_key (address COLLATE NOCASE) WHERE address IS NOT NULL;

CREATE TABLE IF NOT EXISTS latest_round (
	config_name TEXT PRIMARY KEY,
	round       INTEGER NOT NULL
);`

// Round is the latest round of a config, keyed by config name.
type Round struct {
	ConfigName string
	Round      int32
}

// Snapshot is the full store content. Keys are in keyring order.
type Snapshot struct {
	Keys   []utils.SignerKey
	Rounds []Round
}

func (s Snapshot) IsEmpty() bool {
	return len(s.Keys) == 0 && len(s.Rounds) == 0
}

type Store struct {
	db *sql.DB
}

// OpenFromEnv opens the store at the path in NODE_SQLITE_PATH. With
// mustExist, a missing file is an error instead of a new empty store.
func OpenFromEnv(ctx context.Context, mustExist bool) (*Store, error) {
	path, err := sqlitedb.PathFromEnv(EnvPath, mustExist)
	if err != nil {
		return nil, err
	}
	return Open(ctx, path)
}

// Open opens (creating if missing) the SQLite file at path and ensures the schema.
func Open(ctx context.Context, path string) (*Store, error) {
	db, err := sqlitedb.Open(ctx, path, schema)
	if err != nil {
		return nil, err
	}
	return &Store{db: db}, nil
}

func (s *Store) Close() error {
	return s.db.Close()
}

// SignerKeys returns the keyring in insertion order. PK is the encrypted value.
func (s *Store) SignerKeys(ctx context.Context) ([]utils.SignerKey, error) {
	rows, err := s.db.QueryContext(ctx, "SELECT id, address, pk, state, tx_hash, created_at, updated_at FROM signer_key ORDER BY id")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []utils.SignerKey
	for rows.Next() {
		var k utils.SignerKey
		var address, txHash sql.NullString
		var createdAt, updatedAt string
		if err := rows.Scan(&k.ID, &address, &k.PK, &k.State, &txHash, &createdAt, &updatedAt); err != nil {
			return nil, err
		}
		if address.Valid {
			k.Address = &address.String
		}
		if txHash.Valid {
			k.TxHash = &txHash.String
		}
		if k.CreatedAt, err = time.Parse(time.RFC3339Nano, createdAt); err != nil {
			return nil, fmt.Errorf("signer_key %d created_at: %w", k.ID, err)
		}
		if k.UpdatedAt, err = time.Parse(time.RFC3339Nano, updatedAt); err != nil {
			return nil, fmt.Errorf("signer_key %d updated_at: %w", k.ID, err)
		}
		result = append(result, k)
	}
	return result, rows.Err()
}

// LatestRounds returns the latest round per config, ordered by config name.
func (s *Store) LatestRounds(ctx context.Context) ([]Round, error) {
	rows, err := s.db.QueryContext(ctx, "SELECT config_name, round FROM latest_round ORDER BY config_name")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []Round
	for rows.Next() {
		var r Round
		if err := rows.Scan(&r.ConfigName, &r.Round); err != nil {
			return nil, err
		}
		result = append(result, r)
	}
	return result, rows.Err()
}

// Snapshot returns the full store content.
func (s *Store) Snapshot(ctx context.Context) (Snapshot, error) {
	keys, err := s.SignerKeys(ctx)
	if err != nil {
		return Snapshot{}, err
	}
	rounds, err := s.LatestRounds(ctx)
	if err != nil {
		return Snapshot{}, err
	}
	return Snapshot{Keys: keys, Rounds: rounds}, nil
}

// Replace atomically swaps the whole store content for snap. Key IDs in snap
// are ignored; rows get new IDs in slice order.
func (s *Store) Replace(ctx context.Context, snap Snapshot) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	if _, err := tx.ExecContext(ctx, "DELETE FROM signer_key"); err != nil {
		return err
	}
	for _, k := range snap.Keys {
		_, err := tx.ExecContext(ctx,
			"INSERT INTO signer_key (address, pk, state, tx_hash, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?)",
			nullable(k.Address), k.PK, k.State, nullable(k.TxHash),
			k.CreatedAt.UTC().Format(time.RFC3339Nano), k.UpdatedAt.UTC().Format(time.RFC3339Nano))
		if err != nil {
			return fmt.Errorf("insert signer_key: %w", err)
		}
	}
	if err := insertRounds(ctx, tx, snap.Rounds); err != nil {
		return err
	}
	return tx.Commit()
}

// ReplaceRounds atomically swaps all latest rounds, leaving the keyring as is.
func (s *Store) ReplaceRounds(ctx context.Context, rounds []Round) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	if err := insertRounds(ctx, tx, rounds); err != nil {
		return err
	}
	return tx.Commit()
}

func insertRounds(ctx context.Context, tx *sql.Tx, rounds []Round) error {
	if _, err := tx.ExecContext(ctx, "DELETE FROM latest_round"); err != nil {
		return err
	}
	for _, r := range rounds {
		if _, err := tx.ExecContext(ctx, "INSERT INTO latest_round (config_name, round) VALUES (?, ?)", r.ConfigName, r.Round); err != nil {
			return fmt.Errorf("insert latest_round %s: %w", r.ConfigName, err)
		}
	}
	return nil
}

func nullable(s *string) any {
	if s == nil {
		return nil
	}
	return *s
}

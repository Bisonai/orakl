// Package nodestate persists nodebin state (signer keyring + latest round per
// config) in a local SQLite file.
package nodestate

import (
	"context"
	"database/sql"
	"errors"
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

// InsertPendingKey adds a key in state 'pending' and returns its id. pk must
// already be encrypted.
func (s *Store) InsertPendingKey(ctx context.Context, address, encryptedPk string) (int64, error) {
	now := nowText()
	res, err := s.db.ExecContext(ctx,
		"INSERT INTO signer_key (address, pk, state, created_at, updated_at) VALUES (?, ?, 'pending', ?, ?)",
		address, encryptedPk, now, now)
	if err != nil {
		return 0, fmt.Errorf("insert signer_key: %w", err)
	}
	return res.LastInsertId()
}

// BackfillKeyAddress sets the address of a key that has none.
func (s *Store) BackfillKeyAddress(ctx context.Context, id int64, address string) error {
	_, err := s.db.ExecContext(ctx,
		"UPDATE signer_key SET address = ?, updated_at = ? WHERE id = ? AND address IS NULL",
		address, nowText(), id)
	return err
}

// ActivateKey makes key id the single active key, retiring any other active
// key, in one transaction. Re-activating the active key is a no-op; an
// unknown id is an error and changes nothing.
func (s *Store) ActivateKey(ctx context.Context, id int64) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	now := nowText()
	if _, err := tx.ExecContext(ctx,
		"UPDATE signer_key SET state = 'retired', updated_at = ? WHERE state = 'active' AND id <> ?", now, id); err != nil {
		return err
	}
	res, err := tx.ExecContext(ctx,
		"UPDATE signer_key SET state = 'active', updated_at = ? WHERE id = ? AND state <> 'active'", now, id)
	if err != nil {
		return err
	}
	if n, err := res.RowsAffected(); err != nil {
		return err
	} else if n == 0 {
		var exists bool
		if err := tx.QueryRowContext(ctx, "SELECT EXISTS (SELECT 1 FROM signer_key WHERE id = ?)", id).Scan(&exists); err != nil {
			return err
		}
		if !exists {
			return fmt.Errorf("signer_key %d not found", id)
		}
	}
	return tx.Commit()
}

// GCRetiredKeys deletes retired keys last updated before cutoff.
func (s *Store) GCRetiredKeys(ctx context.Context, cutoff time.Time) error {
	keys, err := s.SignerKeys(ctx)
	if err != nil {
		return err
	}
	for _, k := range keys {
		if k.State != "retired" || !k.UpdatedAt.Before(cutoff) {
			continue
		}
		if _, err := s.db.ExecContext(ctx, "DELETE FROM signer_key WHERE id = ? AND state = 'retired'", k.ID); err != nil {
			return err
		}
	}
	return nil
}

// LatestRound returns the stored round of a config; ok is false if none.
func (s *Store) LatestRound(ctx context.Context, configName string) (round int32, ok bool, err error) {
	err = s.db.QueryRowContext(ctx, "SELECT round FROM latest_round WHERE config_name = ?", configName).Scan(&round)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, false, nil
	}
	if err != nil {
		return 0, false, err
	}
	return round, true, nil
}

// SaveRounds upserts rounds in one transaction. A stored round never goes
// down: the higher of stored and given is kept.
func (s *Store) SaveRounds(ctx context.Context, rounds map[string]int32) error {
	if len(rounds) == 0 {
		return nil
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	stmt, err := tx.PrepareContext(ctx,
		"INSERT INTO latest_round (config_name, round) VALUES (?, ?) ON CONFLICT (config_name) DO UPDATE SET round = max(round, excluded.round)")
	if err != nil {
		return err
	}
	defer stmt.Close()
	for name, round := range rounds {
		if _, err := stmt.ExecContext(ctx, name, round); err != nil {
			return fmt.Errorf("save latest_round %s: %w", name, err)
		}
	}
	return tx.Commit()
}

func nowText() string {
	return time.Now().UTC().Format(time.RFC3339Nano)
}

func nullable(s *string) any {
	if s == nil {
		return nil
	}
	return *s
}

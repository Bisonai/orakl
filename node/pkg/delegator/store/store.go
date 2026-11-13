// Package store is the delegator's SQLite state: the signing allowlist in
// config.db and signed-transaction records in one file per UTC day
// (transactions-YYYYMMDD.db), so old records are removed by deleting files.
package store

import (
	"context"
	"database/sql"
	"encoding/csv"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"

	"bisonai.com/miko/node/pkg/utils/sqlitedb"
	"github.com/kaiachain/kaia/crypto"
	"github.com/rs/zerolog/log"
)

const (
	// EnvDir is the env var holding the directory of the SQLite files.
	EnvDir = "DELEGATOR_SQLITE_DIR"
	// EnvRetentionDays is how many daily transaction files to keep (0: all).
	EnvRetentionDays = "DELEGATOR_TX_RETENTION_DAYS"

	DefaultRetentionDays = 7

	configFile = "config.db"
	txPrefix   = "transactions-"
	txSuffix   = ".db"
	dayLayout  = "20060102"
)

const configSchema = `
CREATE TABLE IF NOT EXISTS allowed (
	contract    TEXT NOT NULL,
	selector    TEXT NOT NULL,
	description TEXT NOT NULL DEFAULT '',
	PRIMARY KEY (contract, selector)
);`

const txSchema = `
CREATE TABLE IF NOT EXISTS transactions (
	id        INTEGER PRIMARY KEY AUTOINCREMENT,
	timestamp TEXT NOT NULL,
	sender    TEXT NOT NULL,
	contract  TEXT NOT NULL,
	selector  TEXT NOT NULL,
	nonce     INTEGER NOT NULL,
	tx_hash   TEXT NOT NULL
);`

const upsertQuery = `INSERT INTO allowed (contract, selector, description) VALUES (?, ?, ?)
ON CONFLICT(contract, selector) DO UPDATE SET description = excluded.description`

var (
	addressPattern  = regexp.MustCompile(`^0x[0-9a-f]{40}$`)
	selectorPattern = regexp.MustCompile(`^0x[0-9a-f]{8}$`)
	txFilePattern   = regexp.MustCompile(`^` + txPrefix + `(\d{8})` + txSuffix + `$`)
)

// Entry is one allowed (contract, function selector) pair.
type Entry struct {
	Contract    string
	Selector    string
	Description string
}

// Transaction is the record kept for each signed transaction.
type Transaction struct {
	Timestamp time.Time
	Sender    string
	Contract  string
	Selector  string
	Nonce     uint64
	TxHash    string
}

type Store struct {
	dir           string
	retentionDays int
	config        *sql.DB

	txMu  sync.Mutex
	txDay string
	txDB  *sql.DB
}

// DirFromEnv returns the directory in DELEGATOR_SQLITE_DIR ("" if unset).
func DirFromEnv() string {
	return os.Getenv(EnvDir)
}

// RetentionFromEnv returns DELEGATOR_TX_RETENTION_DAYS, or the default.
func RetentionFromEnv() (int, error) {
	raw := os.Getenv(EnvRetentionDays)
	if raw == "" {
		return DefaultRetentionDays, nil
	}
	var days int
	if _, err := fmt.Sscanf(raw, "%d", &days); err != nil || days < 0 {
		return 0, fmt.Errorf("%s=%q: want a non-negative integer", EnvRetentionDays, raw)
	}
	return days, nil
}

// Open opens dir/config.db, creating the directory and file if missing (a
// missing config.db only when create is true).
func Open(ctx context.Context, dir string, retentionDays int, create bool) (*Store, error) {
	if dir == "" {
		return nil, errors.New(EnvDir + " not set")
	}
	path := filepath.Join(dir, configFile)
	if !create {
		if _, err := os.Stat(path); err != nil {
			return nil, fmt.Errorf("allowlist %s: %w (create it with 'allow add' or 'allow import')", path, err)
		}
	} else if err := os.MkdirAll(dir, 0o700); err != nil {
		return nil, err
	}
	db, err := sqlitedb.Open(ctx, path, configSchema)
	if err != nil {
		return nil, err
	}
	return &Store{dir: dir, retentionDays: retentionDays, config: db}, nil
}

func (s *Store) Close() error {
	s.txMu.Lock()
	defer s.txMu.Unlock()
	if s.txDB != nil {
		_ = s.txDB.Close()
		s.txDB = nil
	}
	return s.config.Close()
}

// NormalizeEntry lowercases contract and selector and turns a function
// signature (e.g. "submit(bytes32[],int256[],uint256[],bytes[])") into its
// 4-byte selector.
func NormalizeEntry(contract, function string) (string, string, error) {
	contract = strings.ToLower(strings.TrimSpace(contract))
	if !addressPattern.MatchString(contract) {
		return "", "", fmt.Errorf("invalid contract address %q", contract)
	}
	function = strings.TrimSpace(function)
	if strings.Contains(function, "(") {
		hash := crypto.Keccak256([]byte(function))
		return contract, "0x" + hex.EncodeToString(hash[:4]), nil
	}
	function = strings.ToLower(function)
	if !selectorPattern.MatchString(function) {
		return "", "", fmt.Errorf("invalid function %q: want 0x-prefixed 4-byte selector or a signature", function)
	}
	return contract, function, nil
}

// ContractAllowed reports whether any function of contract is allowed.
func (s *Store) ContractAllowed(ctx context.Context, contract string) (bool, error) {
	var n int
	err := s.config.QueryRowContext(ctx, "SELECT COUNT(*) FROM allowed WHERE contract = ?", strings.ToLower(contract)).Scan(&n)
	return n > 0, err
}

// FunctionAllowed reports whether selector is allowed on contract.
func (s *Store) FunctionAllowed(ctx context.Context, contract, selector string) (bool, error) {
	var n int
	err := s.config.QueryRowContext(ctx, "SELECT COUNT(*) FROM allowed WHERE contract = ? AND selector = ?",
		strings.ToLower(contract), strings.ToLower(selector)).Scan(&n)
	return n > 0, err
}

// Upsert allows function (selector or signature) on contract.
func (s *Store) Upsert(ctx context.Context, contract, function, description string) (Entry, error) {
	contract, selector, err := NormalizeEntry(contract, function)
	if err != nil {
		return Entry{}, err
	}
	if _, err := s.config.ExecContext(ctx, upsertQuery, contract, selector, description); err != nil {
		return Entry{}, err
	}
	return Entry{Contract: contract, Selector: selector, Description: description}, nil
}

// Remove disallows function on contract; reports whether a row was removed.
func (s *Store) Remove(ctx context.Context, contract, function string) (bool, error) {
	contract, selector, err := NormalizeEntry(contract, function)
	if err != nil {
		return false, err
	}
	res, err := s.config.ExecContext(ctx, "DELETE FROM allowed WHERE contract = ? AND selector = ?", contract, selector)
	if err != nil {
		return false, err
	}
	n, err := res.RowsAffected()
	return n > 0, err
}

// List returns the allowlist ordered by contract and selector.
func (s *Store) List(ctx context.Context) ([]Entry, error) {
	rows, err := s.config.QueryContext(ctx, "SELECT contract, selector, description FROM allowed ORDER BY contract, selector")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []Entry
	for rows.Next() {
		var e Entry
		if err := rows.Scan(&e.Contract, &e.Selector, &e.Description); err != nil {
			return nil, err
		}
		result = append(result, e)
	}
	return result, rows.Err()
}

// ImportCSV upserts "contract,function[,description]" rows (no header) in one
// transaction; never removes entries missing from r. Returns rows imported.
func (s *Store) ImportCSV(ctx context.Context, r io.Reader) (int, error) {
	reader := csv.NewReader(r)
	reader.FieldsPerRecord = -1

	tx, err := s.config.BeginTx(ctx, nil)
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
		if len(record) < 2 {
			return 0, fmt.Errorf("line %d: want contract,function[,description]", n+1)
		}
		contract, selector, err := NormalizeEntry(record[0], record[1])
		if err != nil {
			return 0, fmt.Errorf("line %d: %w", n+1, err)
		}
		description := ""
		if len(record) > 2 {
			description = record[2]
		}
		if _, err := tx.ExecContext(ctx, upsertQuery, contract, selector, description); err != nil {
			return 0, err
		}
		n++
	}
	return n, tx.Commit()
}

// RecordTransaction appends t to the file of its UTC day, switching files at
// midnight and then deleting files older than the retention.
func (s *Store) RecordTransaction(ctx context.Context, t Transaction) error {
	s.txMu.Lock()
	defer s.txMu.Unlock()

	day := t.Timestamp.UTC().Format(dayLayout)
	if s.txDB == nil || day != s.txDay {
		db, err := sqlitedb.Open(ctx, filepath.Join(s.dir, txPrefix+day+txSuffix), txSchema)
		if err != nil {
			return err
		}
		if s.txDB != nil {
			_ = s.txDB.Close()
		}
		s.txDB, s.txDay = db, day
		s.pruneLocked()
	}

	_, err := s.txDB.ExecContext(ctx,
		"INSERT INTO transactions (timestamp, sender, contract, selector, nonce, tx_hash) VALUES (?, ?, ?, ?, ?, ?)",
		t.Timestamp.UTC().Format(time.RFC3339Nano), strings.ToLower(t.Sender), strings.ToLower(t.Contract),
		strings.ToLower(t.Selector), int64(t.Nonce), t.TxHash)
	return err
}

// TransactionFiles returns the daily transaction files, oldest first.
func (s *Store) TransactionFiles() ([]string, error) {
	entries, err := os.ReadDir(s.dir)
	if err != nil {
		return nil, err
	}
	var files []string
	for _, e := range entries {
		if !e.IsDir() && txFilePattern.MatchString(e.Name()) {
			files = append(files, e.Name())
		}
	}
	sort.Strings(files)
	return files, nil
}

// pruneLocked deletes daily files (with their -wal/-shm) older than the
// retention; the current day's file is never deleted. Caller holds txMu.
func (s *Store) pruneLocked() {
	if s.retentionDays <= 0 {
		return
	}
	files, err := s.TransactionFiles()
	if err != nil {
		log.Warn().Err(err).Str("Player", "Delegator").Msg("failed to list transaction files")
		return
	}
	today, err := time.Parse(dayLayout, s.txDay)
	if err != nil {
		return
	}
	cutoff := today.AddDate(0, 0, -(s.retentionDays - 1))
	for _, name := range files {
		day, err := time.Parse(dayLayout, txFilePattern.FindStringSubmatch(name)[1])
		if err != nil || !day.Before(cutoff) {
			continue
		}
		for _, suffix := range []string{"", "-wal", "-shm"} {
			if err := os.Remove(filepath.Join(s.dir, name+suffix)); err != nil && !errors.Is(err, os.ErrNotExist) {
				log.Warn().Err(err).Str("file", name+suffix).Msg("failed to delete old transaction file")
			}
		}
		log.Info().Str("Player", "Delegator").Str("file", name).Msg("deleted old transaction file")
	}
}

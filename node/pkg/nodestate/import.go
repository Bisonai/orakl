package nodestate

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"bisonai.com/miko/node/pkg/chain/utils"
	"bisonai.com/miko/node/pkg/utils/encryptor"
	"github.com/jackc/pgx/v5"
)

const (
	loadLegacySignerQuery = "SELECT pk FROM signer WHERE pk IS NOT NULL AND pk <> '' LIMIT 1"
	loadLatestRoundsQuery = `SELECT c.name AS config_name, MAX(g.round) AS round
		FROM global_aggregates g JOIN configs c ON c.id = g.config_id
		GROUP BY c.name ORDER BY c.name`
)

// ErrStoreNotEmpty is returned by Import when the store holds different
// content and force is not set.
var ErrStoreNotEmpty = errors.New("sqlite state store is not empty and differs from Postgres (use -force to overwrite)")

// Querier is satisfied by *pgxpool.Pool, *pgx.Conn and pgx.Tx.
type Querier interface {
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
}

type ImportResult struct {
	Snapshot  Snapshot
	Unchanged bool     // store already held the same content; nothing written
	Warnings  []string // never contain key material
}

// ReadPostgres reads the signer keyring (+ the legacy `signer` row if it is
// not in the keyring yet) and MAX(round) per config name. Encrypted pks are
// copied as-is; they are decrypted only to derive addresses for matching the
// legacy row against the keyring (needs ENCRYPT_PASSWORD).
func ReadPostgres(ctx context.Context, q Querier) (Snapshot, []string, error) {
	var warnings []string

	rows, err := q.Query(ctx, utils.LOAD_SIGNER_KEYS)
	if err != nil {
		return Snapshot{}, nil, fmt.Errorf("load signer_key: %w", err)
	}
	keys, err := pgx.CollectRows(rows, pgx.RowToStructByName[utils.SignerKey])
	if err != nil {
		return Snapshot{}, nil, fmt.Errorf("load signer_key: %w", err)
	}

	legacyRows, err := q.Query(ctx, loadLegacySignerQuery)
	if err != nil {
		return Snapshot{}, nil, fmt.Errorf("load signer: %w", err)
	}
	legacyPks, err := pgx.CollectRows(legacyRows, pgx.RowTo[string])
	if err != nil {
		return Snapshot{}, nil, fmt.Errorf("load signer: %w", err)
	}

	if len(legacyPks) > 0 {
		legacy, warning := legacyKey(keys, legacyPks[0], time.Now().UTC())
		if warning != "" {
			warnings = append(warnings, warning)
		}
		if legacy != nil {
			keys = append(keys, *legacy)
		}
	}

	roundRows, err := q.Query(ctx, loadLatestRoundsQuery)
	if err != nil {
		return Snapshot{}, nil, fmt.Errorf("load latest rounds: %w", err)
	}
	rounds, err := pgx.CollectRows(roundRows, func(row pgx.CollectableRow) (Round, error) {
		var r Round
		err := row.Scan(&r.ConfigName, &r.Round)
		return r, err
	})
	if err != nil {
		return Snapshot{}, nil, fmt.Errorf("load latest rounds: %w", err)
	}

	return Snapshot{Keys: keys, Rounds: rounds}, warnings, nil
}

// legacyKey returns the legacy signer row as a keyring entry, or nil if the
// keyring already holds it. It is 'active' only if the keyring has no active
// key (as migration 000041 seeds it), else 'pending' so it stays a candidate.
func legacyKey(keys []utils.SignerKey, legacyPk string, now time.Time) (*utils.SignerKey, string) {
	known := map[string]bool{}
	hasActive := false
	for _, k := range keys {
		if k.PK == legacyPk {
			return nil, ""
		}
		if k.State == "active" {
			hasActive = true
		}
		if k.Address != nil && *k.Address != "" {
			known[strings.ToLower(*k.Address)] = true
		} else if addr, ok := deriveAddress(k.PK); ok {
			known[addr] = true
		}
	}

	state := "pending"
	if !hasActive {
		state = "active"
	}
	key := &utils.SignerKey{PK: legacyPk, State: state, CreatedAt: now, UpdatedAt: now}

	addr, ok := deriveAddress(legacyPk)
	if !ok {
		return key, "legacy signer pk could not be decrypted (check ENCRYPT_PASSWORD); imported as " + state + " without address"
	}
	if known[addr] {
		return nil, ""
	}
	key.Address = &addr
	return key, ""
}

// deriveAddress decrypts an encrypted pk and returns its lowercased address.
func deriveAddress(encryptedPk string) (string, bool) {
	if len(encryptedPk) <= 64 { // iv + salt hex prefix; DecryptText slices without checking
		return "", false
	}
	pk, err := encryptor.DecryptText(encryptedPk)
	if err != nil {
		return "", false
	}
	addr, err := utils.StringPkToAddressHex(pk)
	if err != nil {
		return "", false
	}
	return strings.ToLower(addr), true
}

// Import copies Postgres state into store. An empty store, or one with the
// same content, is written/left as is; otherwise it refuses unless force.
func Import(ctx context.Context, q Querier, store *Store, force bool) (ImportResult, error) {
	snap, warnings, err := ReadPostgres(ctx, q)
	if err != nil {
		return ImportResult{}, err
	}
	result, err := importSnapshot(ctx, store, snap, force)
	result.Warnings = warnings
	return result, err
}

func importSnapshot(ctx context.Context, store *Store, snap Snapshot, force bool) (ImportResult, error) {
	result := ImportResult{Snapshot: snap}

	current, err := store.Snapshot(ctx)
	if err != nil {
		return ImportResult{}, err
	}
	if !current.IsEmpty() {
		if sameContent(current, snap) {
			result.Unchanged = true
			return result, nil
		}
		if !force {
			return ImportResult{}, ErrStoreNotEmpty
		}
	}

	if err := store.Replace(ctx, snap); err != nil {
		return ImportResult{}, err
	}
	return result, nil
}

// sameContent compares keys (order, address, pk, state, tx_hash) and rounds.
// IDs and timestamps are ignored: the legacy row gets the import time.
func sameContent(a, b Snapshot) bool {
	if len(a.Keys) != len(b.Keys) || len(a.Rounds) != len(b.Rounds) {
		return false
	}
	for i := range a.Keys {
		x, y := a.Keys[i], b.Keys[i]
		if x.PK != y.PK || x.State != y.State || !equalPtr(x.Address, y.Address) || !equalPtr(x.TxHash, y.TxHash) {
			return false
		}
	}
	for i := range a.Rounds {
		if a.Rounds[i] != b.Rounds[i] {
			return false
		}
	}
	return true
}

func equalPtr(a, b *string) bool {
	if a == nil || b == nil {
		return a == b
	}
	return *a == *b
}

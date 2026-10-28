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

// Ordering must be deterministic and match the store's (id / binary name
// order), since sameContent compares position by position.
const (
	loadSignerKeysQuery   = "SELECT id, address, pk, state, tx_hash, created_at, updated_at FROM signer_key ORDER BY created_at, id"
	loadLegacySignerQuery = "SELECT pk FROM signer WHERE pk IS NOT NULL AND pk <> '' LIMIT 1"
	loadLatestRoundsQuery = `SELECT c.name AS config_name, MAX(g.round) AS round
		FROM global_aggregates g JOIN configs c ON c.id = g.config_id
		GROUP BY c.name ORDER BY c.name COLLATE "C"`
)

// ErrStoreNotEmpty is returned by Import when the store holds a different
// keyring and force is not set.
var ErrStoreNotEmpty = errors.New("sqlite state store holds a different signer keyring than Postgres (use -force to overwrite)")

// Querier is satisfied by *pgxpool.Pool, *pgx.Conn and pgx.Tx.
type Querier interface {
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
}

type ImportResult struct {
	Snapshot   Snapshot
	Unchanged  bool // store already held the same content; nothing written
	RoundsOnly bool // keyring unchanged; only rounds were updated
}

// ReadPostgres reads the signer keyring (+ the legacy `signer` row if it is
// not in the keyring yet) and MAX(round) per config name. Encrypted pks are
// copied as-is; they are decrypted only to derive addresses for matching the
// legacy row against the keyring (needs ENCRYPT_PASSWORD).
func ReadPostgres(ctx context.Context, q Querier) (Snapshot, error) {
	rows, err := q.Query(ctx, loadSignerKeysQuery)
	if err != nil {
		return Snapshot{}, fmt.Errorf("load signer_key: %w", err)
	}
	keys, err := pgx.CollectRows(rows, pgx.RowToStructByName[utils.SignerKey])
	if err != nil {
		return Snapshot{}, fmt.Errorf("load signer_key: %w", err)
	}

	legacyRows, err := q.Query(ctx, loadLegacySignerQuery)
	if err != nil {
		return Snapshot{}, fmt.Errorf("load signer: %w", err)
	}
	legacyPks, err := pgx.CollectRows(legacyRows, pgx.RowTo[string])
	if err != nil {
		return Snapshot{}, fmt.Errorf("load signer: %w", err)
	}

	if len(legacyPks) > 0 {
		legacy, err := legacyKey(keys, legacyPks[0], time.Now().UTC())
		if err != nil {
			return Snapshot{}, err
		}
		if legacy != nil {
			keys = append(keys, *legacy)
		}
	}

	roundRows, err := q.Query(ctx, loadLatestRoundsQuery)
	if err != nil {
		return Snapshot{}, fmt.Errorf("load latest rounds: %w", err)
	}
	rounds, err := pgx.CollectRows(roundRows, func(row pgx.CollectableRow) (Round, error) {
		var r Round
		err := row.Scan(&r.ConfigName, &r.Round)
		return r, err
	})
	if err != nil {
		return Snapshot{}, fmt.Errorf("load latest rounds: %w", err)
	}

	return Snapshot{Keys: keys, Rounds: rounds}, nil
}

// legacyKey returns the legacy signer row as a keyring entry, or nil if the
// keyring already holds it. It is 'active' only if the keyring is empty (as
// migration 000041 seeds it), else 'pending' so it stays a candidate and does
// not outrank an in-flight pending key. A legacy pk that cannot be decrypted
// is an error: likely a wrong ENCRYPT_PASSWORD, so keyring matching is blind.
func legacyKey(keys []utils.SignerKey, legacyPk string, now time.Time) (*utils.SignerKey, error) {
	for _, k := range keys {
		if k.PK == legacyPk {
			return nil, nil
		}
	}

	addr, ok := deriveAddress(legacyPk)
	if !ok {
		return nil, errors.New("legacy signer pk could not be decrypted (check ENCRYPT_PASSWORD)")
	}
	// addresses first; decrypt (scrypt) NULL-address rows only if still needed
	for _, k := range keys {
		if k.Address != nil && strings.EqualFold(*k.Address, addr) {
			return nil, nil
		}
	}
	for _, k := range keys {
		if k.Address == nil || *k.Address == "" {
			if a, ok := deriveAddress(k.PK); ok && a == addr {
				return nil, nil
			}
		}
	}

	state := "pending"
	if len(keys) == 0 {
		state = "active"
	}
	return &utils.SignerKey{Address: &addr, PK: legacyPk, State: state, CreatedAt: now, UpdatedAt: now}, nil
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

// Import copies Postgres state into store. An empty store is written. A
// store with the same keyring is left as is, or gets its rounds updated if
// they differ. A store with a different keyring is refused unless force.
func Import(ctx context.Context, q Querier, store *Store, force bool) (ImportResult, error) {
	snap, err := ReadPostgres(ctx, q)
	if err != nil {
		return ImportResult{}, err
	}
	return importSnapshot(ctx, store, snap, force)
}

func importSnapshot(ctx context.Context, store *Store, snap Snapshot, force bool) (ImportResult, error) {
	result := ImportResult{Snapshot: snap}

	current, err := store.Snapshot(ctx)
	if err != nil {
		return ImportResult{}, err
	}
	if !current.IsEmpty() {
		switch {
		case sameKeys(current.Keys, snap.Keys) && sameRounds(current.Rounds, snap.Rounds):
			result.Unchanged = true
			return result, nil
		case sameKeys(current.Keys, snap.Keys):
			result.RoundsOnly = true
		case !force:
			return ImportResult{}, ErrStoreNotEmpty
		}
	}

	if err := store.Replace(ctx, snap); err != nil {
		return ImportResult{}, err
	}
	return result, nil
}

// sameContent compares keys and rounds.
func sameContent(a, b Snapshot) bool {
	return sameKeys(a.Keys, b.Keys) && sameRounds(a.Rounds, b.Rounds)
}

// sameKeys compares order, address, pk, state and tx_hash. IDs and
// timestamps are ignored: the legacy row gets the import time.
func sameKeys(a, b []utils.SignerKey) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		x, y := a[i], b[i]
		if x.PK != y.PK || x.State != y.State || !equalPtr(x.Address, y.Address) || !equalPtr(x.TxHash, y.TxHash) {
			return false
		}
	}
	return true
}

func sameRounds(a, b []Round) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
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

package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"time"

	"bisonai.com/miko/node/pkg/nodestate"
	"bisonai.com/miko/node/pkg/secrets"
	"bisonai.com/miko/node/pkg/utils/sqlitedb"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

const stateUsage = `usage: nodebin state <command>

Node state (signer keyring + latest round per config) is stored in the SQLite
file at $NODE_SQLITE_PATH. nodebin refuses to start if the file is missing.

commands:
  init              create an empty store for a fresh node (refused if the
                    file exists); the signer keyring is then seeded from
                    SIGNER_PK on first start
  import [-force] [-round-margin N]
                    copy signer keys + MAX(round) per config from Postgres
                    ($DATABASE_URL) into SQLite (import creates the file).
                    Encrypted pks are copied as-is; ENCRYPT_PASSWORD is only
                    used to match the legacy signer row against the keyring.
                    A store with the same keyring only gets its rounds
                    advanced; a different keyring, or stored rounds newer
                    than or missing from Postgres, is refused unless -force,
                    which replaces the whole store. -round-margin adds N
                    to every imported round, to skip rounds a still-running
                    node uses after the import (Postgres rounds also lag).
  list              print signer key addresses + states and latest rounds
                    (never prints private keys)
`

func runState(ctx context.Context, args []string, stdout io.Writer) error {
	if len(args) == 0 {
		return fmt.Errorf("%s", stateUsage)
	}

	switch args[0] {
	case "import":
		fs := flag.NewFlagSet("import", flag.ContinueOnError)
		fs.SetOutput(io.Discard)
		force := fs.Bool("force", false, "replace a store with a different keyring or newer rounds")
		roundMargin := fs.Int("round-margin", 0, "add N to every imported round")
		if err := fs.Parse(args[1:]); err != nil || fs.NArg() != 0 || *roundMargin < 0 || *roundMargin > maxRoundMargin {
			return fmt.Errorf("%s", stateUsage)
		}
		return runStateImport(ctx, *force, int32(*roundMargin), stdout)
	case "init":
		if len(args) != 1 {
			return fmt.Errorf("%s", stateUsage)
		}
		return runStateInit(ctx, stdout)
	case "list":
		if len(args) != 1 {
			return fmt.Errorf("%s", stateUsage)
		}
		return runStateList(ctx, stdout)
	default:
		return fmt.Errorf("%s", stateUsage)
	}
}

// maxRoundMargin caps -round-margin (1M rounds = ~5 days at 400ms).
const maxRoundMargin = 1_000_000

func runStateImport(ctx context.Context, force bool, roundMargin int32, stdout io.Writer) error {
	// fail on a missing env before touching Postgres; create the file only
	// after Postgres was read, so a failed read leaves no empty store behind
	if os.Getenv(nodestate.EnvPath) == "" {
		return fmt.Errorf("%s environment variable not set", nodestate.EnvPath)
	}

	// own pool (not db.GetPool) so a bad DATABASE_URL errors out with a timeout
	dsn := secrets.GetSecret("DATABASE_URL")
	if dsn == "" {
		return errors.New("DATABASE_URL not set")
	}
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		return fmt.Errorf("connect postgres: %w", err)
	}
	defer pool.Close()
	pingCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	if err := pool.Ping(pingCtx); err != nil {
		return fmt.Errorf("connect postgres: %w", err)
	}

	// one consistent snapshot of keyring, legacy row and rounds
	tx, err := pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.RepeatableRead, AccessMode: pgx.ReadOnly})
	if err != nil {
		return err
	}
	snap, err := nodestate.ReadPostgres(ctx, tx)
	_ = tx.Rollback(ctx)
	if err != nil {
		return err
	}
	addRoundMargin(snap.Rounds, roundMargin)

	store, err := nodestate.OpenFromEnv(ctx, false)
	if err != nil {
		return err
	}
	defer store.Close()

	result, err := nodestate.Apply(ctx, store, snap, force)
	if err != nil {
		return err
	}
	keys, rounds := len(result.Snapshot.Keys), len(result.Snapshot.Rounds)
	switch {
	case result.Unchanged:
		fmt.Fprintf(stdout, "unchanged: store already has %d signer keys, %d rounds\n", keys, rounds)
	case result.RoundsOnly:
		fmt.Fprintf(stdout, "signer keys unchanged (%d); updated %d rounds\n", keys, rounds)
	default:
		fmt.Fprintf(stdout, "imported %d signer keys, %d rounds\n", keys, rounds)
	}
	return nil
}

func runStateInit(ctx context.Context, stdout io.Writer) error {
	path, err := sqlitedb.PathFromEnv(nodestate.EnvPath, false)
	if err != nil {
		return err
	}
	if _, err := os.Stat(path); err == nil {
		return fmt.Errorf("%s already exists", path)
	} else if !errors.Is(err, os.ErrNotExist) {
		return err
	}
	store, err := nodestate.Open(ctx, path)
	if err != nil {
		return err
	}
	if err := store.Close(); err != nil {
		return err
	}
	fmt.Fprintf(stdout, "created empty store %s\n", path)
	return nil
}

func addRoundMargin(rounds []nodestate.Round, margin int32) {
	for i := range rounds {
		rounds[i].Round += margin
	}
}

// runStateList prints key metadata and rounds; pk is never printed.
func runStateList(ctx context.Context, stdout io.Writer) error {
	// a missing file is a mistake (wrong path, unmounted volume)
	store, err := nodestate.OpenFromEnv(ctx, true)
	if err != nil {
		return err
	}
	defer store.Close()

	snap, err := store.Snapshot(ctx)
	if err != nil {
		return err
	}

	fmt.Fprintln(stdout, "signer keys (address, state, tx_hash, updated_at):")
	for _, k := range snap.Keys {
		fmt.Fprintf(stdout, "%s\t%s\t%s\t%s\n", deref(k.Address), k.State, deref(k.TxHash), k.UpdatedAt.Format(time.RFC3339))
	}
	fmt.Fprintln(stdout, "latest rounds (config, round):")
	for _, r := range snap.Rounds {
		fmt.Fprintf(stdout, "%s\t%d\n", r.ConfigName, r.Round)
	}
	return nil
}

func deref(s *string) string {
	if s == nil || *s == "" {
		return "-"
	}
	return *s
}

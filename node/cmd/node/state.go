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
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

const stateUsage = `usage: nodebin state <command>

Node state (signer keyring + latest round per config) is stored in the SQLite
file at $NODE_SQLITE_PATH. nodebin does not use it at runtime yet.

commands:
  import [-force]   copy signer keys + MAX(round) per config from Postgres
                    ($DATABASE_URL) into SQLite (import creates the file).
                    Encrypted pks are copied as-is; ENCRYPT_PASSWORD is only
                    used to match the legacy signer row against the keyring.
                    A store with the same keyring only gets its rounds
                    advanced; a different keyring, or stored rounds newer
                    than or missing from Postgres, is refused unless -force,
                    which replaces the whole store.
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
		if err := fs.Parse(args[1:]); err != nil || fs.NArg() != 0 {
			return fmt.Errorf("%s", stateUsage)
		}
		return runStateImport(ctx, *force, stdout)
	case "list":
		if len(args) != 1 {
			return fmt.Errorf("%s", stateUsage)
		}
		return runStateList(ctx, stdout)
	default:
		return fmt.Errorf("%s", stateUsage)
	}
}

func runStateImport(ctx context.Context, force bool, stdout io.Writer) error {
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

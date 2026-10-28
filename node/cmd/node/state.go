package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"time"

	"bisonai.com/miko/node/pkg/db"
	"bisonai.com/miko/node/pkg/nodestate"
)

const stateUsage = `usage: nodebin state <command>

Node state (signer keyring + latest round per config) is stored in the SQLite
file at $NODE_SQLITE_PATH. nodebin does not use it at runtime yet.

commands:
  import [-force]   copy signer keys + MAX(round) per config from Postgres
                    ($DATABASE_URL) into SQLite (import creates the file).
                    Encrypted pks are copied as-is; ENCRYPT_PASSWORD is only
                    used to match the legacy signer row against the keyring.
                    A store with the same content is left untouched; a store
                    with different content is refused unless -force, which
                    replaces it.
  list              print signer key addresses + states and latest rounds
                    (never prints private keys)
`

func runState(ctx context.Context, args []string, stdout, stderr io.Writer) error {
	if len(args) == 0 {
		return fmt.Errorf("%s", stateUsage)
	}

	switch args[0] {
	case "import":
		fs := flag.NewFlagSet("import", flag.ContinueOnError)
		fs.SetOutput(io.Discard)
		force := fs.Bool("force", false, "overwrite a non-empty store")
		if err := fs.Parse(args[1:]); err != nil || fs.NArg() != 0 {
			return fmt.Errorf("%s", stateUsage)
		}
		return runStateImport(ctx, *force, stdout, stderr)
	case "list":
		if len(args) != 1 {
			return fmt.Errorf("%s", stateUsage)
		}
		return runStateList(ctx, stdout)
	default:
		return fmt.Errorf("%s", stateUsage)
	}
}

func runStateImport(ctx context.Context, force bool, stdout, stderr io.Writer) error {
	pool, err := db.GetPool(ctx)
	if err != nil {
		return fmt.Errorf("connect postgres: %w", err)
	}

	store, err := nodestate.OpenFromEnv(ctx, false)
	if err != nil {
		return err
	}
	defer store.Close()

	result, err := nodestate.Import(ctx, pool, store, force)
	if err != nil {
		return err
	}
	for _, w := range result.Warnings {
		fmt.Fprintln(stderr, "warning:", w)
	}
	if result.Unchanged {
		fmt.Fprintf(stdout, "unchanged: store already has %d signer keys, %d rounds\n", len(result.Snapshot.Keys), len(result.Snapshot.Rounds))
		return nil
	}
	fmt.Fprintf(stdout, "imported %d signer keys, %d rounds\n", len(result.Snapshot.Keys), len(result.Snapshot.Rounds))
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

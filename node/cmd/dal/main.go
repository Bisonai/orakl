package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"

	"bisonai.com/miko/node/pkg/dal"
	"bisonai.com/miko/node/pkg/dal/utils/keystore"
	"bisonai.com/miko/node/pkg/utils/loginit"
	"github.com/joho/godotenv"
	"github.com/rs/zerolog/log"
)

const keysUsage = `usage: dal [-env file] keys <command>

API keys are stored in the SQLite file at $DAL_SQLITE_PATH.

commands:
  list                         print all keys as "key<TAB>description"
  add <key> <description>      add a key, or update its description
  remove <key>                 remove a key
  import <file.csv|->          upsert "key,description" CSV rows (no header)

Changes are picked up immediately for new keys; a removed key can stay
valid in a running DAL for up to the key cache TTL (1h) unless DAL is restarted.
`

func LoadEnvFromFile() {
	envFile := flag.String("env", "", "env file")
	flag.Parse()

	if *envFile != "" {
		log.Info().Msgf("loading env file: %s", *envFile)
		err := godotenv.Load(*envFile)
		if err != nil {
			panic(err)
		}
	}
}

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	LoadEnvFromFile()

	if args := flag.Args(); len(args) > 0 && args[0] == "keys" {
		if err := runKeys(ctx, args[1:], os.Stdin, os.Stdout); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		return
	}

	loginit.InitZeroLog()

	err := dal.Run(ctx)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to start DAL")
	}
}

func runKeys(ctx context.Context, args []string, stdin io.Reader, stdout io.Writer) error {
	if len(args) == 0 {
		return fmt.Errorf("%s", keysUsage)
	}

	store, err := keystore.OpenFromEnv(ctx)
	if err != nil {
		return err
	}
	defer store.Close()

	switch cmd := args[0]; {
	case cmd == "list" && len(args) == 1:
		keys, err := store.List(ctx)
		if err != nil {
			return err
		}
		for _, k := range keys {
			fmt.Fprintf(stdout, "%s\t%s\n", k.Key, k.Description)
		}
		return nil
	case cmd == "add" && len(args) == 3:
		if err := store.Upsert(ctx, args[1], args[2]); err != nil {
			return err
		}
		fmt.Fprintf(stdout, "added key (description %q)\n", args[2])
		return nil
	case cmd == "remove" && len(args) == 2:
		removed, err := store.Remove(ctx, args[1])
		if err != nil {
			return err
		}
		if !removed {
			return fmt.Errorf("key not found")
		}
		fmt.Fprintln(stdout, "removed key")
		return nil
	case cmd == "import" && len(args) == 2:
		in := stdin
		if args[1] != "-" {
			f, err := os.Open(args[1])
			if err != nil {
				return err
			}
			defer f.Close()
			in = f
		}
		n, err := store.ImportCSV(ctx, in)
		if err != nil {
			return err
		}
		fmt.Fprintf(stdout, "imported %d keys\n", n)
		return nil
	default:
		return fmt.Errorf("%s", keysUsage)
	}
}

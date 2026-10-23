package main

import (
	"context"
	"fmt"
	"io"
	"os"

	"bisonai.com/miko/node/pkg/boot"
	"bisonai.com/miko/node/pkg/boot/utils/peerstore"
	"bisonai.com/miko/node/pkg/utils/loginit"
	"github.com/rs/zerolog/log"
)

const peersUsage = `usage: bootbin peers <command>

Peers are stored in the SQLite file at $BOOT_SQLITE_PATH.

commands:
  list                print all peers as "id<TAB>url"
  import <file|->     add one peer url per line, skipping known urls;
                      creates the file if missing

Nodes register themselves through POST /api/v1/peer/sync, and the boot API
removes unreachable peers every refresh interval, so import is only needed
to seed a new file.
`

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if len(os.Args) > 1 && os.Args[1] == "peers" {
		if err := runPeers(ctx, os.Args[2:], os.Stdin, os.Stdout); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		return
	}

	loginit.InitZeroLog()

	err := boot.Run(ctx)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to start bootnode")
		return
	}
}

func runPeers(ctx context.Context, args []string, stdin io.Reader, stdout io.Writer) error {
	wantArgs := map[string]int{"list": 1, "import": 2}
	if len(args) == 0 || wantArgs[args[0]] != len(args) {
		return fmt.Errorf("%s", peersUsage)
	}

	// only import may create the file; list on a missing file is a mistake (wrong path, unmounted volume)
	store, err := peerstore.OpenFromEnv(ctx, args[0] != "import")
	if err != nil {
		return err
	}
	defer store.Close()

	switch args[0] {
	case "list":
		peers, err := store.List(ctx)
		if err != nil {
			return err
		}
		for _, p := range peers {
			fmt.Fprintf(stdout, "%d\t%s\n", p.ID, p.Url)
		}
		return nil
	default: // import
		in := stdin
		if args[1] != "-" {
			f, err := os.Open(args[1])
			if err != nil {
				return err
			}
			defer f.Close()
			in = f
		}
		n, err := store.ImportURLs(ctx, in)
		if err != nil {
			return err
		}
		fmt.Fprintf(stdout, "imported %d new peers\n", n)
		return nil
	}
}

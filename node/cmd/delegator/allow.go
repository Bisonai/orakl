package main

import (
	"context"
	"fmt"
	"io"
	"os"

	"bisonai.com/miko/node/pkg/delegator/store"
)

const allowUsage = `usage: delegatorbin allow <command>

SQLite mode only. The allowlist is in $DELEGATOR_SQLITE_DIR/config.db
(add/import create it). <function> is a 0x-prefixed 4-byte selector or a
signature such as "submit(bytes32[],int256[],uint256[],bytes[])".

commands:
  list                                    print "contract<TAB>selector<TAB>description"
  add <contract> <function> [description] allow function on contract
  remove <contract> <function>            disallow function on contract
  import <file.csv|->                     upsert "contract,function[,description]"
                                          rows (no header); never removes entries

A running delegator caches approvals: added entries apply at once, removed
ones after a restart.
`

func runAllow(ctx context.Context, args []string, stdin io.Reader, stdout io.Writer) error {
	if len(args) == 0 {
		return fmt.Errorf("%s", allowUsage)
	}
	cmd := args[0]
	valid := (cmd == "list" && len(args) == 1) ||
		(cmd == "add" && (len(args) == 3 || len(args) == 4)) ||
		(cmd == "remove" && len(args) == 3) ||
		(cmd == "import" && len(args) == 2)
	if !valid {
		return fmt.Errorf("%s", allowUsage)
	}

	// only add/import may create the file; list/remove on a missing file is a
	// mistake (wrong path, unmounted volume)
	create := cmd == "add" || cmd == "import"
	s, err := store.Open(ctx, store.DirFromEnv(), 0, create)
	if err != nil {
		return err
	}
	defer s.Close()

	switch cmd {
	case "list":
		entries, err := s.List(ctx)
		if err != nil {
			return err
		}
		for _, e := range entries {
			fmt.Fprintf(stdout, "%s\t%s\t%s\n", e.Contract, e.Selector, e.Description)
		}
	case "add":
		description := ""
		if len(args) == 4 {
			description = args[3]
		}
		e, err := s.Upsert(ctx, args[1], args[2], description)
		if err != nil {
			return err
		}
		fmt.Fprintf(stdout, "allowed %s on %s\n", e.Selector, e.Contract)
	case "remove":
		removed, err := s.Remove(ctx, args[1], args[2])
		if err != nil {
			return err
		}
		if !removed {
			return fmt.Errorf("entry not found")
		}
		fmt.Fprintln(stdout, "removed entry")
	case "import":
		in := stdin
		if args[1] != "-" {
			f, err := os.Open(args[1])
			if err != nil {
				return err
			}
			defer f.Close()
			in = f
		}
		n, err := s.ImportCSV(ctx, in)
		if err != nil {
			return err
		}
		fmt.Fprintf(stdout, "imported %d entries\n", n)
	}
	return nil
}

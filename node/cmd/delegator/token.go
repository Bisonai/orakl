package main

import (
	"context"
	"fmt"
	"io"

	"bisonai.com/miko/node/pkg/delegator/store"
)

const tokenUsage = `usage: delegatorbin token <command>

SQLite mode only. Tokens are in $DELEGATOR_SQLITE_DIR/config.db (add creates
it); only their SHA-256 is stored. Every /sign request needs
"Authorization: Bearer <token>"; with no tokens all are refused.

commands:
  list           print "name<TAB>created_at"
  add <name>     create a token and print it once (keep it in the client's secret store)
  remove <name>  revoke a token (applies to the next request)
`

func runToken(ctx context.Context, args []string, stdout io.Writer) error {
	if len(args) == 0 {
		return fmt.Errorf("%s", tokenUsage)
	}
	cmd := args[0]
	valid := (cmd == "list" && len(args) == 1) ||
		(cmd == "add" && len(args) == 2) ||
		(cmd == "remove" && len(args) == 2)
	if !valid {
		return fmt.Errorf("%s", tokenUsage)
	}

	s, err := store.Open(ctx, store.DirFromEnv(), 0, cmd == "add")
	if err != nil {
		return err
	}
	defer s.Close()

	switch cmd {
	case "list":
		tokens, err := s.ListTokens(ctx)
		if err != nil {
			return err
		}
		for _, t := range tokens {
			fmt.Fprintf(stdout, "%s\t%s\n", t.Name, t.CreatedAt)
		}
	case "add":
		token, err := s.AddToken(ctx, args[1])
		if err != nil {
			return err
		}
		fmt.Fprintln(stdout, token)
	case "remove":
		removed, err := s.RemoveToken(ctx, args[1])
		if err != nil {
			return err
		}
		if !removed {
			return fmt.Errorf("token not found")
		}
		fmt.Fprintln(stdout, "removed token")
	}
	return nil
}

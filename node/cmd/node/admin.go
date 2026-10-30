package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"bisonai.com/miko/node/pkg/admin/utils"
)

const adminUsage = `usage: nodebin admin [-timeout 2m] <command> [args]

Calls the admin API of the node running in this pod/host
(http://127.0.0.1:$APP_PORT, APP_PORT defaults to 8088). Sends
"Authorization: Bearer <ADMIN_TOKEN>" when ADMIN_TOKEN is set (Vault or env),
no header otherwise; with ADMIN_AUTH_REQUIRED=true an empty token is an error
and nothing is sent. Prints HTTP status + response body; exits non-zero on a
non-2xx response or connection error. -timeout applies per call; a timed-out
call may still complete on the node.

commands:
  config-sync                     POST /api/v1/config/sync
  fetcher-start                   POST /api/v1/fetcher/start
  fetcher-stop                    POST /api/v1/fetcher/stop
  fetcher-refresh                 POST /api/v1/fetcher/refresh
  aggregator-start                POST /api/v1/aggregator/start
  aggregator-stop                 POST /api/v1/aggregator/stop
  aggregator-refresh              POST /api/v1/aggregator/refresh
  aggregator-activate <id>        POST /api/v1/aggregator/activate/<id>
  aggregator-deactivate <id>      POST /api/v1/aggregator/deactivate/<id>
  renew-signer                    POST /api/v1/aggregator/renew-signer
  host-sync                       POST /api/v1/host/sync
  reload                          config-sync, fetcher-refresh,
                                  aggregator-refresh; stops at first failure
  help                            print this usage
`

const (
	adminTokenEnv     = utils.AdminTokenEnv
	adminCallTimeout  = 2 * time.Minute // default -timeout
	adminMaxBodyBytes = 1 << 20
)

// adminRoutes maps commands without arguments to admin API paths.
var adminRoutes = map[string]string{
	"config-sync":        "/api/v1/config/sync",
	"fetcher-start":      "/api/v1/fetcher/start",
	"fetcher-stop":       "/api/v1/fetcher/stop",
	"fetcher-refresh":    "/api/v1/fetcher/refresh",
	"aggregator-start":   "/api/v1/aggregator/start",
	"aggregator-stop":    "/api/v1/aggregator/stop",
	"aggregator-refresh": "/api/v1/aggregator/refresh",
	"renew-signer":       "/api/v1/aggregator/renew-signer",
	"host-sync":          "/api/v1/host/sync",
}

// adminIDRoutes maps commands taking a config ID to admin API path prefixes.
var adminIDRoutes = map[string]string{
	"aggregator-activate":   "/api/v1/aggregator/activate/",
	"aggregator-deactivate": "/api/v1/aggregator/deactivate/",
}

var adminReloadSteps = []string{"config-sync", "fetcher-refresh", "aggregator-refresh"}

// errAdminUsage marks invalid invocations (exit code 2).
var errAdminUsage = errors.New(adminUsage)

// adminPort is the admin API port, same default as pkg/admin (see #2722).
func adminPort() string {
	if port := os.Getenv("APP_PORT"); port != "" {
		return port
	}
	return "8088"
}

type adminClient struct {
	baseURL string
	token   string
	// tokenErr fails every call (ADMIN_AUTH_REQUIRED set but token empty)
	tokenErr error
	timeout  time.Duration
	http     *http.Client
	stdout   io.Writer
}

func newAdminClient(stdout io.Writer) *adminClient {
	token, tokenErr := utils.AdminToken()
	return &adminClient{
		baseURL:  "http://127.0.0.1:" + adminPort(),
		token:    token,
		tokenErr: tokenErr,
		timeout:  adminCallTimeout,
		http: &http.Client{
			// a redirect is unexpected; don't follow it (and forward the token)
			CheckRedirect: func(*http.Request, []*http.Request) error {
				return http.ErrUseLastResponse
			},
		},
		stdout: stdout,
	}
}

func runAdmin(ctx context.Context, args []string, stdout io.Writer) error {
	return newAdminClient(stdout).run(ctx, args)
}

// adminExitCode maps a runAdmin error to the process exit code.
func adminExitCode(err error) int {
	switch {
	case err == nil:
		return 0
	case errors.Is(err, errAdminUsage):
		return 2
	default:
		return 1
	}
}

func (c *adminClient) run(ctx context.Context, args []string) error {
	fs := flag.NewFlagSet("admin", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	timeout := fs.Duration("timeout", c.timeout, "per-call timeout")
	// flags are accepted before and after the command
	var positional []string
	for {
		if err := fs.Parse(args); err != nil {
			if errors.Is(err, flag.ErrHelp) && len(positional) == 0 && len(fs.Args()) == 0 {
				fmt.Fprint(c.stdout, adminUsage)
				return nil
			}
			return fmt.Errorf("%v: %w", err, errAdminUsage)
		}
		rest := fs.Args()
		// "--" ends flag parsing; everything after it is positional
		if n := len(args) - len(rest); n > 0 && args[n-1] == "--" {
			positional = append(positional, rest...)
			break
		}
		if len(rest) == 0 {
			break
		}
		positional = append(positional, rest[0])
		args = rest[1:]
	}
	if *timeout <= 0 {
		return fmt.Errorf("invalid -timeout %s: %w", *timeout, errAdminUsage)
	}
	c.timeout = *timeout
	args = positional
	if len(args) == 0 {
		return errAdminUsage
	}
	cmd := args[0]

	if cmd == "help" {
		if len(args) != 1 {
			return errAdminUsage
		}
		fmt.Fprint(c.stdout, adminUsage)
		return nil
	}

	if cmd == "reload" {
		if len(args) != 1 {
			return errAdminUsage
		}
		if c.tokenErr != nil {
			return fmt.Errorf("reload: %w", c.tokenErr)
		}
		for i, step := range adminReloadSteps {
			if err := c.call(ctx, step, adminRoutes[step]); err != nil {
				return fmt.Errorf("reload stopped, not run: %s: %w", strings.Join(adminReloadSteps[i+1:], ", "), err)
			}
		}
		return nil
	}

	if path, ok := adminRoutes[cmd]; ok {
		if len(args) != 1 {
			return errAdminUsage
		}
		return c.call(ctx, cmd, path)
	}

	if prefix, ok := adminIDRoutes[cmd]; ok {
		if len(args) != 2 {
			return errAdminUsage
		}
		id, err := strconv.ParseInt(args[1], 10, 32)
		if err != nil || id < 1 {
			return fmt.Errorf("invalid config id %q: %w", args[1], errAdminUsage)
		}
		return c.call(ctx, cmd, prefix+strconv.FormatInt(id, 10))
	}

	return errAdminUsage
}

// call POSTs to path and prints the status and body; the token is never printed.
func (c *adminClient) call(ctx context.Context, name, path string) error {
	if c.tokenErr != nil {
		return fmt.Errorf("%s: %w", name, c.tokenErr)
	}
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	url := c.baseURL + path
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, nil)
	if err != nil {
		return fmt.Errorf("%s: %w", name, err)
	}
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}

	resp, err := c.http.Do(req)
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			return fmt.Errorf("%s: POST %s: timed out after %s (may still complete on the node): %w", name, url, c.timeout, err)
		}
		return fmt.Errorf("%s: POST %s: %w", name, url, err)
	}
	defer resp.Body.Close()

	// status first: it tells whether the node accepted the command even if
	// reading the body fails
	fmt.Fprintf(c.stdout, "%s: POST %s: %s\n", name, path, resp.Status)
	body, err := io.ReadAll(io.LimitReader(resp.Body, adminMaxBodyBytes+1))
	truncated := len(body) > adminMaxBodyBytes
	if truncated {
		body = body[:adminMaxBodyBytes]
	}
	if len(body) > 0 {
		fmt.Fprintln(c.stdout, strings.TrimRight(string(body), "\n"))
	}
	if truncated {
		fmt.Fprintf(c.stdout, "(response truncated at %d bytes)\n", adminMaxBodyBytes)
	}
	ok := resp.StatusCode >= 200 && resp.StatusCode <= 299
	if err != nil {
		if ok {
			// the node accepted the command; only the body is incomplete
			fmt.Fprintf(c.stdout, "(read response: %v)\n", err)
			return nil
		}
		return fmt.Errorf("%s: %s, read response: %w", name, resp.Status, err)
	}
	if !ok {
		return fmt.Errorf("%s: %s", name, resp.Status)
	}
	return nil
}

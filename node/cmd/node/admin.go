package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"bisonai.com/miko/node/pkg/secrets"
)

const adminUsage = `usage: nodebin admin <command>

Calls the admin API of the node running in this pod/host
(http://127.0.0.1:$APP_PORT, APP_PORT defaults to 8088). Sends
"Authorization: Bearer <ADMIN_TOKEN>" when ADMIN_TOKEN is set (Vault or env),
no header otherwise. Prints HTTP status + response body; exits non-zero on a
non-2xx response or connection error.

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
`

const (
	adminTokenEnv     = "ADMIN_TOKEN"
	adminCallTimeout  = 2 * time.Minute
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
	http    *http.Client
	stdout  io.Writer
}

func newAdminClient(stdout io.Writer) *adminClient {
	return &adminClient{
		baseURL: "http://127.0.0.1:" + adminPort(),
		token:   strings.TrimSpace(secrets.GetSecret(adminTokenEnv)),
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
	if len(args) == 0 {
		return errAdminUsage
	}
	cmd := args[0]

	if cmd == "reload" {
		if len(args) != 1 {
			return errAdminUsage
		}
		for _, step := range adminReloadSteps {
			if err := c.call(ctx, step, adminRoutes[step]); err != nil {
				return fmt.Errorf("reload stopped: %w", err)
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
	ctx, cancel := context.WithTimeout(ctx, adminCallTimeout)
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
		return fmt.Errorf("%s: POST %s: %w", name, url, err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, adminMaxBodyBytes))
	if err != nil {
		return fmt.Errorf("%s: read response: %w", name, err)
	}

	fmt.Fprintf(c.stdout, "%s: POST %s: %s\n", name, path, resp.Status)
	if len(body) > 0 {
		fmt.Fprintln(c.stdout, strings.TrimRight(string(body), "\n"))
	}
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return fmt.Errorf("%s: %s", name, resp.Status)
	}
	return nil
}

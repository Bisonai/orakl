package main

import (
	"bytes"
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testAdminToken = "s3cret-admin-token"

type adminRequest struct {
	method, path, auth string
	hasAuth            bool
}

// fakeAdmin records requests; paths in fail answer 500.
type fakeAdmin struct {
	mu   sync.Mutex
	reqs []adminRequest
	fail map[string]bool
}

func (f *fakeAdmin) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	f.mu.Lock()
	_, hasAuth := r.Header["Authorization"]
	f.reqs = append(f.reqs, adminRequest{r.Method, r.URL.Path, r.Header.Get("Authorization"), hasAuth})
	fail := f.fail[r.URL.Path]
	f.mu.Unlock()
	if fail {
		http.Error(w, "boom", http.StatusInternalServerError)
		return
	}
	_, _ = w.Write([]byte("ok " + r.URL.Path))
}

func (f *fakeAdmin) requests() []adminRequest {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]adminRequest(nil), f.reqs...)
}

func (f *fakeAdmin) paths() []string {
	var out []string
	for _, r := range f.requests() {
		out = append(out, r.path)
	}
	return out
}

// startFakeAdmin points APP_PORT at a local fake admin server and sets ADMIN_TOKEN.
func startFakeAdmin(t *testing.T, token string, fail ...string) *fakeAdmin {
	t.Helper()
	f := &fakeAdmin{fail: map[string]bool{}}
	for _, p := range fail {
		f.fail[p] = true
	}
	srv := httptest.NewServer(f)
	t.Cleanup(srv.Close)
	u, err := url.Parse(srv.URL)
	require.NoError(t, err)
	t.Setenv("APP_PORT", u.Port())
	t.Setenv(adminTokenEnv, token)
	return f
}

func TestAdminCommandsRoutes(t *testing.T) {
	cases := map[string]string{
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
	assert.Len(t, adminRoutes, len(cases))
	for cmd, path := range cases {
		f := startFakeAdmin(t, "")
		var out bytes.Buffer
		require.NoError(t, runAdmin(context.Background(), []string{cmd}, &out), cmd)
		reqs := f.requests()
		require.Len(t, reqs, 1, cmd)
		assert.Equal(t, http.MethodPost, reqs[0].method, cmd)
		assert.Equal(t, path, reqs[0].path, cmd)
		assert.Contains(t, out.String(), "200 OK", cmd)
		assert.Contains(t, out.String(), "ok "+path, cmd)
	}
}

func TestAdminActivateDeactivate(t *testing.T) {
	f := startFakeAdmin(t, "")
	var out bytes.Buffer
	require.NoError(t, runAdmin(context.Background(), []string{"aggregator-activate", "7"}, &out))
	require.NoError(t, runAdmin(context.Background(), []string{"aggregator-deactivate", "12"}, &out))
	assert.Equal(t, []string{"/api/v1/aggregator/activate/7", "/api/v1/aggregator/deactivate/12"}, f.paths())
}

func TestAdminUsage(t *testing.T) {
	f := startFakeAdmin(t, testAdminToken)
	for _, args := range [][]string{
		{}, {"bogus"}, {"reload", "x"}, {"config-sync", "x"},
		{"aggregator-activate"}, {"aggregator-activate", "1", "2"},
		{"aggregator-activate", "abc"}, {"aggregator-activate", "0"}, {"aggregator-activate", "-1"},
		{"aggregator-deactivate", "../config/sync"}, {"aggregator-deactivate", "2147483648"},
		{"aggregator-deactivate", "1.5"},
		{"-timeout", "0", "config-sync"}, {"-timeout", "x", "config-sync"}, {"-nope", "config-sync"},
	} {
		var out bytes.Buffer
		err := runAdmin(context.Background(), args, &out)
		assert.ErrorContains(t, err, "usage: nodebin admin", args)
		assert.Equal(t, 2, adminExitCode(err), args)
	}
	assert.Empty(t, f.paths(), "invalid invocations must not call the API")
}

func TestAdminAuthHeader(t *testing.T) {
	t.Run("token set", func(t *testing.T) {
		f := startFakeAdmin(t, testAdminToken)
		require.NoError(t, runAdmin(context.Background(), []string{"host-sync"}, &bytes.Buffer{}))
		reqs := f.requests()
		require.Len(t, reqs, 1)
		assert.Equal(t, "Bearer "+testAdminToken, reqs[0].auth)
	})
	t.Run("token with whitespace", func(t *testing.T) {
		f := startFakeAdmin(t, " "+testAdminToken+"\n")
		require.NoError(t, runAdmin(context.Background(), []string{"host-sync"}, &bytes.Buffer{}))
		reqs := f.requests()
		require.Len(t, reqs, 1)
		assert.Equal(t, "Bearer "+testAdminToken, reqs[0].auth)
	})
	for name, token := range map[string]string{"token empty": "", "token blank": " \n"} {
		t.Run(name, func(t *testing.T) {
			f := startFakeAdmin(t, token)
			require.NoError(t, runAdmin(context.Background(), []string{"host-sync"}, &bytes.Buffer{}))
			reqs := f.requests()
			require.Len(t, reqs, 1)
			assert.False(t, reqs[0].hasAuth, "no Authorization header without token")
		})
	}
}

func TestAdminNon2xx(t *testing.T) {
	f := startFakeAdmin(t, testAdminToken, "/api/v1/fetcher/refresh")
	var out bytes.Buffer
	err := runAdmin(context.Background(), []string{"fetcher-refresh"}, &out)
	require.Error(t, err)
	assert.Equal(t, 1, adminExitCode(err))
	assert.Contains(t, err.Error(), "500")
	assert.Contains(t, out.String(), "500 Internal Server Error")
	assert.Contains(t, out.String(), "boom")
	assert.NotContains(t, out.String()+err.Error(), testAdminToken)
	assert.Len(t, f.paths(), 1)
}

func TestAdminConnectionError(t *testing.T) {
	// privileged port 1 (tcpmux): nothing listens there on test hosts
	t.Setenv("APP_PORT", "1")
	t.Setenv(adminTokenEnv, testAdminToken)

	var out bytes.Buffer
	err := runAdmin(context.Background(), []string{"config-sync"}, &out)
	require.Error(t, err)
	assert.Equal(t, 1, adminExitCode(err))
	assert.False(t, errors.Is(err, errAdminUsage))
	assert.NotContains(t, out.String()+err.Error(), testAdminToken)
}

func TestAdminReload(t *testing.T) {
	t.Run("all steps in order", func(t *testing.T) {
		f := startFakeAdmin(t, testAdminToken)
		var out bytes.Buffer
		require.NoError(t, runAdmin(context.Background(), []string{"reload"}, &out))
		assert.Equal(t, []string{"/api/v1/config/sync", "/api/v1/fetcher/refresh", "/api/v1/aggregator/refresh"}, f.paths())
		for _, r := range f.requests() {
			assert.Equal(t, "Bearer "+testAdminToken, r.auth)
		}
		assert.NotContains(t, out.String(), testAdminToken)
	})
	t.Run("stops at first failure", func(t *testing.T) {
		f := startFakeAdmin(t, testAdminToken, "/api/v1/fetcher/refresh")
		var out bytes.Buffer
		err := runAdmin(context.Background(), []string{"reload"}, &out)
		require.Error(t, err)
		assert.Equal(t, 1, adminExitCode(err))
		assert.Contains(t, err.Error(), "fetcher-refresh")
		assert.Equal(t, []string{"/api/v1/config/sync", "/api/v1/fetcher/refresh"}, f.paths())
		assert.NotContains(t, out.String()+err.Error(), testAdminToken)
	})
}

func TestAdminDefaultPort(t *testing.T) {
	t.Setenv("APP_PORT", "")
	assert.Equal(t, "8088", adminPort())
	t.Setenv("APP_PORT", "9000")
	assert.Equal(t, "9000", adminPort())
}

func TestAdminHelp(t *testing.T) {
	f := startFakeAdmin(t, testAdminToken)
	for _, args := range [][]string{{"help"}, {"--help"}, {"-h"}} {
		var out bytes.Buffer
		require.NoError(t, runAdmin(context.Background(), args, &out), args)
		assert.Contains(t, out.String(), "usage: nodebin admin", args)
	}
	assert.Empty(t, f.paths())
}

func TestAdminTimeoutFlagPosition(t *testing.T) {
	f := startFakeAdmin(t, "")
	for _, args := range [][]string{
		{"-timeout", "5s", "reload"}, {"reload", "-timeout", "5s"},
		{"aggregator-activate", "-timeout=5s", "3"}, {"aggregator-activate", "3", "-timeout", "5s"},
	} {
		require.NoError(t, runAdmin(context.Background(), args, &bytes.Buffer{}), args)
	}
	assert.Len(t, f.paths(), 2*3+2)
}

func TestAdminTimeout(t *testing.T) {
	release := make(chan struct{})
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		<-release
	}))
	t.Cleanup(srv.Close)
	t.Cleanup(func() { close(release) })
	u, err := url.Parse(srv.URL)
	require.NoError(t, err)
	t.Setenv("APP_PORT", u.Port())
	t.Setenv(adminTokenEnv, testAdminToken)

	var out bytes.Buffer
	err = runAdmin(context.Background(), []string{"-timeout", "50ms", "fetcher-refresh"}, &out)
	require.Error(t, err)
	assert.Equal(t, 1, adminExitCode(err))
	assert.Contains(t, err.Error(), "timed out after 50ms")
	assert.NotContains(t, out.String()+err.Error(), testAdminToken)
}

func TestAdminBodyTruncated(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write(bytes.Repeat([]byte("a"), adminMaxBodyBytes+10))
	}))
	t.Cleanup(srv.Close)
	u, err := url.Parse(srv.URL)
	require.NoError(t, err)
	t.Setenv("APP_PORT", u.Port())

	var out bytes.Buffer
	require.NoError(t, runAdmin(context.Background(), []string{"host-sync"}, &out))
	assert.Contains(t, out.String(), "(response truncated at")
	assert.Less(t, out.Len(), adminMaxBodyBytes+200)
}

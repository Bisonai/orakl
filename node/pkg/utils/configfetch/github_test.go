package configfetch

import (
	"context"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fakeGitHub serves the token and contents endpoints. It verifies the App JWT
// with pub and answers contents requests carrying a valid token with bundle.
type fakeGitHub struct {
	srv      *httptest.Server
	mints    atomic.Int32
	gets     atomic.Int32
	reject   atomic.Int32 // number of upcoming contents requests to answer 401
	tokenTTL time.Duration
}

func newFakeGitHub(t *testing.T, pub *rsa.PublicKey, clientID, installationID, bundle string) *fakeGitHub {
	t.Helper()
	f := &fakeGitHub{tokenTTL: time.Hour}
	var current atomic.Value
	current.Store("")
	f.srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/app/installations/"+installationID+"/access_tokens":
			jwt := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
			if !verifyJWT(t, pub, jwt, clientID) {
				w.WriteHeader(http.StatusUnauthorized)
				return
			}
			var req struct {
				Repositories []string          `json:"repositories"`
				Permissions  map[string]string `json:"permissions"`
			}
			_ = json.NewDecoder(r.Body).Decode(&req)
			assert.Equal(t, []string{"orakl-config"}, req.Repositories)
			assert.Equal(t, map[string]string{"contents": "read"}, req.Permissions)
			n := f.mints.Add(1)
			tok := "tok-" + string(rune('0'+n))
			current.Store(tok)
			w.WriteHeader(http.StatusCreated)
			_ = json.NewEncoder(w).Encode(map[string]any{"token": tok, "expires_at": time.Now().Add(f.tokenTTL)})
		case r.Method == http.MethodGet && r.URL.Path == "/repos/Bisonai/orakl-config/contents/mainnet_feeds.json":
			f.gets.Add(1)
			if f.reject.Load() > 0 {
				f.reject.Add(-1)
				w.WriteHeader(http.StatusUnauthorized)
				return
			}
			if r.Header.Get("Authorization") != "Bearer "+current.Load().(string) || r.URL.Query().Get("ref") != "master" ||
				r.Header.Get("Accept") != "application/vnd.github.raw+json" {
				w.WriteHeader(http.StatusNotFound)
				return
			}
			_, _ = w.Write([]byte(bundle))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	t.Cleanup(f.srv.Close)
	return f
}

func verifyJWT(t *testing.T, pub *rsa.PublicKey, jwt, clientID string) bool {
	t.Helper()
	parts := strings.Split(jwt, ".")
	if len(parts) != 3 {
		return false
	}
	sig, err := base64.RawURLEncoding.DecodeString(parts[2])
	if err != nil {
		return false
	}
	sum := sha256.Sum256([]byte(parts[0] + "." + parts[1]))
	if rsa.VerifyPKCS1v15(pub, crypto.SHA256, sum[:], sig) != nil {
		return false
	}
	raw, _ := base64.RawURLEncoding.DecodeString(parts[1])
	var c struct {
		Iat, Exp int64
		Iss      string
	}
	if json.Unmarshal(raw, &c) != nil {
		return false
	}
	now := time.Now().Unix()
	return c.Iss == clientID && c.Iat < now && c.Exp > now && c.Exp-c.Iat <= 600
}

// writeKey writes a fresh RSA key as PEM (PKCS#1 or PKCS#8) and returns its path.
func writeKey(t *testing.T, pkcs8 bool) (string, *rsa.PrivateKey) {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	block := &pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(key)}
	if pkcs8 {
		der, err := x509.MarshalPKCS8PrivateKey(key)
		require.NoError(t, err)
		block = &pem.Block{Type: "PRIVATE KEY", Bytes: der}
	}
	path := filepath.Join(t.TempDir(), "app.pem")
	require.NoError(t, os.WriteFile(path, pem.EncodeToMemory(block), 0o600))
	return path, key
}

// useGitHub configures the App env against f and resets the token cache.
func useGitHub(t *testing.T, f *fakeGitHub, keyFile string) {
	t.Helper()
	orig := githubAPI
	githubAPI = f.srv.URL
	t.Cleanup(func() { githubAPI = orig })
	appsMu.Lock()
	apps = map[string]*githubApp{}
	appsMu.Unlock()
	t.Setenv("CONFIG_GITHUB_APP_CLIENT_ID", "Iv1.test")
	t.Setenv("CONFIG_GITHUB_INSTALLATION_ID", "123")
	t.Setenv("CONFIG_GITHUB_APP_KEY_FILE", keyFile)
	t.Setenv("CONFIG_GITHUB_REF", "")
}

func TestGitHubAppFetchCachesToken(t *testing.T) {
	for _, pkcs8 := range []bool{false, true} {
		fastPolicy(t)
		keyFile, key := writeKey(t, pkcs8)
		f := newFakeGitHub(t, &key.PublicKey, "Iv1.test", "123", validBundle)
		useGitHub(t, f, keyFile)

		for i := 0; i < 3; i++ {
			got, err := Get[[]map[string]any](context.Background(), FeedsPath("mainnet"))
			require.NoError(t, err)
			assert.Equal(t, "BTC-USDT", got[0]["name"])
		}
		assert.EqualValues(t, 1, f.mints.Load(), "token is reused")
		assert.EqualValues(t, 3, f.gets.Load())
	}
}

func TestGitHubAppRenewsNearExpiry(t *testing.T) {
	fastPolicy(t)
	keyFile, key := writeKey(t, false)
	f := newFakeGitHub(t, &key.PublicKey, "Iv1.test", "123", validBundle)
	f.tokenTTL = tokenRenewBefore - time.Second // already inside the renew window
	useGitHub(t, f, keyFile)

	for i := 0; i < 2; i++ {
		_, err := Fetch(context.Background(), FeedsPath("mainnet"))
		require.NoError(t, err)
	}
	assert.EqualValues(t, 2, f.mints.Load())
}

func TestGitHubApp401RemintsToken(t *testing.T) {
	fastPolicy(t)
	keyFile, key := writeKey(t, false)
	f := newFakeGitHub(t, &key.PublicKey, "Iv1.test", "123", validBundle)
	useGitHub(t, f, keyFile)
	f.reject.Store(1)

	_, err := Fetch(context.Background(), FeedsPath("mainnet"))
	require.NoError(t, err)
	assert.EqualValues(t, 2, f.mints.Load(), "401 drops the cached token")
	assert.EqualValues(t, 2, f.gets.Load())
}

func TestGitHubAppWrongKeyFailsFast(t *testing.T) {
	fastPolicy(t)
	_, key := writeKey(t, false)
	otherKeyFile, _ := writeKey(t, false)
	f := newFakeGitHub(t, &key.PublicKey, "Iv1.test", "123", validBundle)
	useGitHub(t, f, otherKeyFile)

	_, err := Fetch(context.Background(), FeedsPath("mainnet"))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "status 401")
	assert.EqualValues(t, 0, f.mints.Load(), "no token minted")
	assert.EqualValues(t, 0, f.gets.Load())
}

func TestGitHubAppSources(t *testing.T) {
	fastPolicy(t)
	keyFile, key := writeKey(t, false)
	f := newFakeGitHub(t, &key.PublicKey, "Iv1.test", "123", validBundle)
	useGitHub(t, f, keyFile)

	// App set: public URL defaults are dropped.
	os.Unsetenv("CONFIG_BASE_URL")
	os.Unsetenv("CONFIG_FALLBACK_URL")
	srcs, err := sources()
	require.NoError(t, err)
	require.Len(t, srcs, 1)
	assert.Equal(t, f.srv.URL+"/repos/Bisonai/orakl-config/contents/mainnet_feeds.json?ref=master", URL(FeedsPath("mainnet")))

	// Explicit URLs are tried after the App source.
	t.Setenv("CONFIG_BASE_URL", "http://a/")
	t.Setenv("CONFIG_FALLBACK_URL", "http://b")
	t.Setenv("CONFIG_GITHUB_REF", "abc123")
	srcs, err = sources()
	require.NoError(t, err)
	require.Len(t, srcs, 3)
	assert.Equal(t, "http://a/x.json", srcs[1].url("/x.json"))
	assert.Equal(t, "http://b/x.json", srcs[2].url("/x.json"))
	assert.Contains(t, srcs[0].url("/x.json"), "?ref=abc123")
}

func TestGitHubAppFallsBackToURL(t *testing.T) {
	fastPolicy(t)
	keyFile, key := writeKey(t, false)
	f := newFakeGitHub(t, &key.PublicKey, "Iv1.test", "123", validBundle)
	useGitHub(t, f, keyFile)
	f.reject.Store(100)
	srv, _ := server(t, body(validBundle))
	t.Setenv("CONFIG_FALLBACK_URL", srv.URL)

	_, err := Fetch(context.Background(), FeedsPath("mainnet"))
	require.NoError(t, err)
}

func TestGitHubAppPartialEnv(t *testing.T) {
	fastPolicy(t)
	t.Setenv("CONFIG_GITHUB_APP_CLIENT_ID", "Iv1.test")
	t.Setenv("CONFIG_GITHUB_INSTALLATION_ID", "")
	t.Setenv("CONFIG_GITHUB_APP_KEY_FILE", "")

	_, err := Fetch(context.Background(), FeedsPath("mainnet"))
	require.ErrorContains(t, err, "must be set together")
}

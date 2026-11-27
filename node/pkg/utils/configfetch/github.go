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
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"sync"
	"time"

	"bisonai.com/orakl/node/pkg/utils/request"
)

// GitHub App source: when CONFIG_GITHUB_APP_CLIENT_ID,
// CONFIG_GITHUB_INSTALLATION_ID and CONFIG_GITHUB_APP_KEY_FILE are all set,
// configs are read from the orakl-config repo through the GitHub contents API
// with a short-lived installation token, and that source is tried first.
// CONFIG_GITHUB_REF (default master) selects the branch, tag or commit.
const (
	githubOwner      = "Bisonai"
	githubRepo       = "orakl-config"
	githubDefaultRef = "master"
	githubAPIVersion = "2022-11-28"
	// A token is renewed this long before GitHub expires it (tokens live 1h).
	tokenRenewBefore = 5 * time.Minute
)

// githubAPI is a package var so tests can point it at a fake server.
var githubAPI = "https://api.github.com"

type githubApp struct {
	clientID       string
	installationID string
	keyFile        string

	mu      sync.Mutex
	token   string
	expires time.Time
}

var (
	appsMu sync.Mutex
	apps   = map[string]*githubApp{} // keyed by config, so the token cache survives env re-reads
)

// githubFromEnv returns the configured App, nil when none is configured, or an
// error when only some of the variables are set.
func githubFromEnv() (*githubApp, error) {
	clientID := os.Getenv("CONFIG_GITHUB_APP_CLIENT_ID")
	installationID := os.Getenv("CONFIG_GITHUB_INSTALLATION_ID")
	keyFile := os.Getenv("CONFIG_GITHUB_APP_KEY_FILE")
	if clientID == "" && installationID == "" && keyFile == "" {
		return nil, nil
	}
	if clientID == "" || installationID == "" || keyFile == "" {
		return nil, errors.New("CONFIG_GITHUB_APP_CLIENT_ID, CONFIG_GITHUB_INSTALLATION_ID and CONFIG_GITHUB_APP_KEY_FILE must be set together")
	}
	k := clientID + "|" + installationID + "|" + keyFile
	appsMu.Lock()
	defer appsMu.Unlock()
	a, ok := apps[k]
	if !ok {
		a = &githubApp{clientID: clientID, installationID: installationID, keyFile: keyFile}
		apps[k] = a
	}
	return a, nil
}

func githubRef() string {
	if ref := os.Getenv("CONFIG_GITHUB_REF"); ref != "" {
		return ref
	}
	return githubDefaultRef
}

// githubContentsURL is the contents API URL of path (which starts with "/").
func githubContentsURL(path string) string {
	return githubAPI + "/repos/" + githubOwner + "/" + githubRepo + "/contents" + path + "?ref=" + githubRef()
}

func (a *githubApp) headers(ctx context.Context) (map[string]string, bool, error) {
	token, retryable, err := a.Token(ctx)
	if err != nil {
		return nil, retryable, err
	}
	return map[string]string{
		"Authorization":        "Bearer " + token,
		"Accept":               "application/vnd.github.raw+json",
		"X-GitHub-Api-Version": githubAPIVersion,
	}, false, nil
}

// invalidate drops the cached token (e.g. after a 401) so the next call mints a
// new one.
func (a *githubApp) invalidate() {
	a.mu.Lock()
	a.token = ""
	a.mu.Unlock()
}

// Token returns a cached installation token, minting a new one when it is
// missing or close to expiry. retryable reports whether a failure is transient.
func (a *githubApp) Token(ctx context.Context) (token string, retryable bool, err error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.token != "" && time.Until(a.expires) > tokenRenewBefore {
		return a.token, false, nil
	}

	jwt, err := a.jwt(time.Now())
	if err != nil {
		return "", false, err
	}
	resp, err := request.RequestRaw(
		request.WithContext(ctx),
		request.WithMethod(http.MethodPost),
		request.WithEndpoint(githubAPI+"/app/installations/"+a.installationID+"/access_tokens"),
		request.WithHeaders(map[string]string{
			"Authorization":        "Bearer " + jwt,
			"Accept":               "application/vnd.github+json",
			"X-GitHub-Api-Version": githubAPIVersion,
		}),
		// Narrow the token to read-only access to orakl-config.
		request.WithBody(map[string]any{
			"repositories": []string{githubRepo},
			"permissions":  map[string]string{"contents": "read"},
		}),
		request.WithTimeout(timeout))
	if err != nil {
		return "", true, fmt.Errorf("github app token: %w", err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 64<<10))
	if err != nil {
		return "", true, fmt.Errorf("github app token: %w", err)
	}
	if resp.StatusCode != http.StatusCreated {
		return "", isRetryableStatus(resp.StatusCode), fmt.Errorf("github app token: status %d", resp.StatusCode)
	}
	var out struct {
		Token     string    `json:"token"`
		ExpiresAt time.Time `json:"expires_at"`
	}
	if err := json.Unmarshal(body, &out); err != nil || out.Token == "" {
		return "", false, fmt.Errorf("github app token: invalid response")
	}
	a.token, a.expires = out.Token, out.ExpiresAt
	return a.token, false, nil
}

// jwt returns the App JWT (RS256) used to request an installation token. The
// key file is read on every mint so a rotated key is picked up without restart.
func (a *githubApp) jwt(now time.Time) (string, error) {
	pemBytes, err := os.ReadFile(a.keyFile)
	if err != nil {
		return "", fmt.Errorf("github app key: %w", err)
	}
	key, err := parseRSAKey(pemBytes)
	if err != nil {
		return "", fmt.Errorf("github app key %s: %w", a.keyFile, err)
	}
	enc := base64.RawURLEncoding
	header := enc.EncodeToString([]byte(`{"alg":"RS256","typ":"JWT"}`))
	claims, err := json.Marshal(map[string]any{
		"iat": now.Add(-60 * time.Second).Unix(), // tolerate clock drift
		"exp": now.Add(9 * time.Minute).Unix(),   // GitHub max is 10 min
		"iss": a.clientID,
	})
	if err != nil {
		return "", err
	}
	signing := header + "." + enc.EncodeToString(claims)
	sum := sha256.Sum256([]byte(signing))
	sig, err := rsa.SignPKCS1v15(rand.Reader, key, crypto.SHA256, sum[:])
	if err != nil {
		return "", err
	}
	return signing + "." + enc.EncodeToString(sig), nil
}

// parseRSAKey accepts PKCS#1 (what GitHub issues) and PKCS#8 PEM keys.
func parseRSAKey(pemBytes []byte) (*rsa.PrivateKey, error) {
	block, _ := pem.Decode(pemBytes)
	if block == nil {
		return nil, errors.New("no PEM block")
	}
	if k, err := x509.ParsePKCS1PrivateKey(block.Bytes); err == nil {
		return k, nil
	}
	k, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		return nil, errors.New("not an RSA private key")
	}
	rk, ok := k.(*rsa.PrivateKey)
	if !ok {
		return nil, errors.New("not an RSA private key")
	}
	return rk, nil
}

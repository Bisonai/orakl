package apiv2

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"

	"bisonai.com/orakl/node/pkg/utils/buildinfo"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func getBody(t *testing.T, url string) (int, string) {
	t.Helper()
	resp, err := http.Get(url)
	require.NoError(t, err)
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	return resp.StatusCode, string(body)
}

// The root text is matched by substring in health checks (KakaoCloud
// deploy.sh, Zabbix): it must keep "Orakl Node DAL API" as an exact prefix.
func TestRootAndVersionWithoutAPIKey(t *testing.T) {
	srv := newTestServer(t)
	before := testutil.CollectAndCount(restRequestsTotal)

	status, body := getBody(t, srv.URL+"/")
	assert.Equal(t, http.StatusOK, status)
	assert.True(t, strings.HasPrefix(body, "Orakl Node DAL API"), body)
	assert.Equal(t, buildinfo.Root("Orakl Node DAL API"), body)

	status, body = getBody(t, srv.URL+"/version")
	assert.Equal(t, http.StatusOK, status)
	var info buildinfo.Info
	require.NoError(t, json.Unmarshal([]byte(body), &info))
	assert.Equal(t, buildinfo.Get(), info)

	assert.Equal(t, before, testutil.CollectAndCount(restRequestsTotal))
}

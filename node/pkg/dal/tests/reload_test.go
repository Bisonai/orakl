//nolint:all
package test

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"bisonai.com/miko/node/pkg/dal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func symbolsFromApi(t *testing.T, testItems *TestItems) []string {
	t.Helper()
	req, err := http.NewRequest(http.MethodGet, testItems.MockDal.URL+"/symbols", nil)
	require.NoError(t, err)
	req.Header.Set("X-API-Key", testItems.ApiKey)
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	var symbols []string
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&symbols))
	return symbols
}

// A symbol added by a config reload is collected from redis and served; a
// removed one is dropped from the API.
func TestReloadAddRemoveSymbol(t *testing.T) {
	ctx := context.Background()
	clean, testItems, err := setup(ctx)
	require.NoError(t, err)
	defer func() {
		if cleanupErr := clean(); cleanupErr != nil {
			t.Logf("Cleanup failed: %v", cleanupErr)
		}
	}()

	const added = "test-aggregate-2"
	err = dal.ApplyConfigs(ctx, testItems.Collector, testItems.Controller, []Config{testItems.TmpConfig, {ID: 14, Name: added}})
	require.NoError(t, err)
	assert.ElementsMatch(t, []string{"test-aggregate", added}, symbolsFromApi(t, testItems))

	data, err := generateSampleSubmissionData(14, 20, time.Now(), 1, added)
	require.NoError(t, err)
	publishAndAwait(ctx, t, testItems, added, *data)
	got, err := testItems.Collector.GetLatestData(added)
	require.NoError(t, err)
	assert.Equal(t, "20", got.Value)

	// remove the original symbol
	err = dal.ApplyConfigs(ctx, testItems.Collector, testItems.Controller, []Config{{ID: 14, Name: added}})
	require.NoError(t, err)
	assert.Equal(t, []string{added}, symbolsFromApi(t, testItems))
	_, err = testItems.Collector.GetLatestData("test-aggregate")
	assert.Error(t, err)

	// still collecting the kept symbol after the resubscribe
	data, err = generateSampleSubmissionData(14, 21, time.Now(), 2, added)
	require.NoError(t, err)
	require.Eventually(t, func() bool {
		if err := testPublishData(ctx, added, *data); err != nil {
			return false
		}
		got, err := testItems.Collector.GetLatestData(added)
		return err == nil && got.Value == "21"
	}, 10*time.Second, 100*time.Millisecond)

	// data for the removed symbol is ignored
	old, err := generateSampleSubmissionData(13, 30, time.Now(), 3, "test-aggregate")
	require.NoError(t, err)
	require.NoError(t, testPublishData(ctx, "test-aggregate", *old))
	time.Sleep(200 * time.Millisecond)
	_, err = testItems.Collector.GetLatestData("test-aggregate")
	assert.Error(t, err)
}

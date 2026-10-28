package nodeconfig

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const bundle = `[
  {"name": "BTC-USDT", "fetchInterval": 2000, "aggregateInterval": 400, "submitInterval": 60000, "feedDataFreshness": 60000,
   "feeds": [
     {"name": "binance-wss-BTC-USDT", "definition": {"type": "wss", "provider": "binance", "base": "BTC", "quote": "USDT"}},
     {"name": "http-BTC-USDT", "definition": {"url": "https://example.com", "reducers": []}},
     {"name": "uni-BTC-USDT", "definition": {"type": "UniswapPool", "chainId": "1", "address": "0x1"}}
   ]},
  {"name": "KRW-USD", "decimals": 16,
   "feeds": [{"name": "http-KRW-USD", "definition": {"url": "https://example.com/krw"}}]}
]`

func parse(t *testing.T, s string) []Entry {
	t.Helper()
	var entries []Entry
	require.NoError(t, json.Unmarshal([]byte(s), &entries))
	return entries
}

func TestURL(t *testing.T) {
	cases := map[string]string{
		"mainnet": "https://config.orakl.network/mainnet_feeds.json",
		"kairos":  "https://config.orakl.network/kairos_feeds.json",
		"":        "https://config.orakl.network/kairos_feeds.json",
	}
	for chain, want := range cases {
		t.Setenv("CHAIN", chain)
		assert.Equal(t, want, URL(), "CHAIN=%q", chain)
	}
}

func TestSetAndQuery(t *testing.T) {
	s := New()
	require.NoError(t, s.Set(parse(t, bundle)))

	configs := s.Configs()
	require.Len(t, configs, 2)
	btc, krw := configs[0], configs[1]
	assert.Equal(t, "BTC-USDT", btc.Name)
	assert.Equal(t, int32(400), btc.AggregateInterval)
	assert.Equal(t, 60000, *btc.FeedDataFreshness)
	assert.Nil(t, btc.Decimals)

	// missing intervals get the defaults
	assert.Equal(t, int32(DefaultFetchInterval), krw.FetchInterval)
	assert.Equal(t, int32(DefaultAggregateInterval), krw.AggregateInterval)
	assert.Equal(t, int32(DefaultSubmitInterval), krw.SubmitInterval)
	assert.Equal(t, 16, *krw.Decimals)

	assert.Len(t, s.Feeds(btc.ID), 3)
	http := s.HTTPFeeds(btc.ID)
	require.Len(t, http, 1)
	assert.Equal(t, "http-BTC-USDT", http[0].Name)
	assert.Equal(t, btc.ID, http[0].ConfigID)

	wss := s.FeedsOfType("wss")
	require.Len(t, wss, 1)
	assert.Equal(t, "binance-wss-BTC-USDT", wss[0].Name)
	require.Len(t, s.FeedsOfType("UniswapPool"), 1)
	assert.Empty(t, s.FeedsOfType("PancakeswapPool"))
}

// IDs are stable per name across syncs and never reused.
func TestSetKeepsIDsStable(t *testing.T) {
	s := New()
	require.NoError(t, s.Set(parse(t, bundle)))
	before := map[string]int32{}
	for _, c := range s.Configs() {
		before[c.Name] = c.ID
	}
	btcFeeds := map[string]int32{}
	for _, f := range s.Feeds(before["BTC-USDT"]) {
		btcFeeds[f.Name] = f.ID
	}

	// drop KRW-USD and one feed, add ETH-USDT, change BTC-USDT interval
	require.NoError(t, s.Set(parse(t, `[
	  {"name": "ETH-USDT", "feeds": [{"name": "http-ETH-USDT", "definition": {"url": "x"}}]},
	  {"name": "BTC-USDT", "aggregateInterval": 500, "feeds": [{"name": "http-BTC-USDT", "definition": {"url": "y"}}]}
	]`)))
	after := map[string]Config{}
	for _, c := range s.Configs() {
		after[c.Name] = c
	}
	require.Len(t, after, 2)
	assert.Equal(t, before["BTC-USDT"], after["BTC-USDT"].ID)
	assert.Equal(t, int32(500), after["BTC-USDT"].AggregateInterval)
	assert.NotEqual(t, before["KRW-USD"], after["ETH-USDT"].ID, "removed IDs are not reused")

	feeds := s.Feeds(after["BTC-USDT"].ID)
	require.Len(t, feeds, 1)
	assert.Equal(t, btcFeeds["http-BTC-USDT"], feeds[0].ID)
	assert.JSONEq(t, `{"url": "y"}`, string(feeds[0].Definition))
	assert.Empty(t, s.Feeds(before["KRW-USD"]))

	// re-adding a removed config gets its old ID back
	require.NoError(t, s.Set(parse(t, bundle)))
	for _, c := range s.Configs() {
		assert.Equal(t, before[c.Name], c.ID, c.Name)
	}
}

func TestSetRejectsInvalidAndKeepsPrevious(t *testing.T) {
	s := New()
	require.NoError(t, s.Set(parse(t, bundle)))

	bad := [][]Entry{
		nil,
		{{Name: ""}},
		{{Name: "A"}, {Name: "A"}},
		{{Name: "A", Feeds: []FeedEntry{{Name: "f", Definition: json.RawMessage(`{}`)}, {Name: "f", Definition: json.RawMessage(`{}`)}}}},
		{{Name: "A", Feeds: []FeedEntry{{Name: "", Definition: json.RawMessage(`{}`)}}}},
		{{Name: "A", Feeds: []FeedEntry{{Name: "f"}}}},
	}
	for i, entries := range bad {
		assert.Error(t, s.Set(entries), "case %d", i)
	}
	assert.Len(t, s.Configs(), 2, "previous content kept")
}

func TestFeedType(t *testing.T) {
	cases := []struct {
		def     string
		typ     string
		hasType bool
	}{
		{`{"type": "wss"}`, "wss", true},
		{`{"type": null}`, "", true},
		{`{"url": "x"}`, "", false},
		{`[]`, "", false},
	}
	for _, c := range cases {
		typ, has := FeedType(Feed{Definition: json.RawMessage(c.def)})
		assert.Equal(t, c.typ, typ, c.def)
		assert.Equal(t, c.hasType, has, c.def)
	}
}

package por

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"bisonai.com/miko/node/pkg/fetcher"
	"github.com/stretchr/testify/assert"
)

// TestMergedConfigUnmarshal verifies that a single merged mag7 config payload
// (the shape served at /mag7/<env>/<name>.json after the config cut-over)
// unmarshals into both the adaptor and the aggregator struct from the same bytes.
func TestMergedConfigUnmarshal(t *testing.T) {
	body := []byte(`{
		"name": "AAPL",
		"decimals": 8,
		"interval": 60000,
		"feeds": [{"name": "aapl-feed", "definition": {"url": "https://example.com"}}],
		"address": "0x1234567890123456789012345678901234567890",
		"heartbeat": 300000,
		"threshold": 0.002,
		"absoluteThreshold": 0.1
	}`)

	var ad adaptor
	assert.NoError(t, json.Unmarshal(body, &ad))
	assert.Equal(t, "AAPL", ad.Name)
	assert.Equal(t, 8, ad.Decimals)
	assert.NotNil(t, ad.Interval)
	assert.Equal(t, 60000, *ad.Interval)
	assert.Len(t, ad.Feeds, 1)
	assert.NotEmpty(t, ad.Feeds[0].Definition)

	var ag aggregator
	assert.NoError(t, json.Unmarshal(body, &ag))
	assert.Equal(t, "AAPL", ag.Name)
	assert.Equal(t, "0x1234567890123456789012345678901234567890", ag.Address)
	assert.NotNil(t, ag.Heartbeat)
	assert.Equal(t, 300000, *ag.Heartbeat)
	assert.Equal(t, 0.002, ag.Threshold)
	assert.Equal(t, 0.1, ag.AbsoluteThreshold)
}

func TestNew(t *testing.T) {
	ctx := context.Background()
	app, err := New(ctx)
	if err != nil {
		t.Fatal(err)
	}

	t.Log(app.entries)
	assert.NotNil(t, app.kaiaHelper)

	publicAddress, err := app.kaiaHelper.PublicAddressString()
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, "0x7a8cD2921BEC42378EAea68f4d0309464d0c50c5", publicAddress)
}

func TestReadLatestRoundId(t *testing.T) {
	ctx := context.Background()
	app, err := New(ctx)
	if err != nil {
		t.Fatal(err)
	}

	roundId, err := app.getRoundId(ctx, app.entries["aapl"])
	if err != nil {
		t.Fatal(err)
	}

	assert.Greater(t, roundId, uint32(0))
}

func TestGetLastInfo(t *testing.T) {
	ctx := context.Background()
	app, err := New(ctx)
	if err != nil {
		t.Fatal(err)
	}
	info, err := app.getLastInfo(ctx, app.entries["aapl"])
	if err != nil {
		t.Fatal(err)
	}
	updatedTime := time.Unix(info.UpdatedAt.Int64(), 0)
	assert.True(t, updatedTime.Before(time.Now()))

	answer := info.Answer.Int64()
	assert.Greater(t, answer, int64(0))
}

func TestExecute(t *testing.T) {
	ctx := context.Background()
	app, err := New(ctx)
	if err != nil {
		t.Fatal(err)
	}
	err = app.execute(ctx, app.entries["aapl"])
	if err != nil {
		t.Fatal(err)
	}
}

func TestReport(t *testing.T) {
	ctx := context.Background()
	app, err := New(ctx)
	if err != nil {
		t.Fatal(err)
	}

	value, err := fetcher.FetchSingle(ctx, app.entries["aapl"].definition)
	if err != nil {
		t.Fatal(err)
	}

	roundId, err := app.getRoundId(ctx, app.entries["aapl"])
	if err != nil {
		t.Fatal(err)
	}

	err = app.report(ctx, app.entries["aapl"], value, roundId)
	if err != nil {
		t.Fatal(err)
	}
}

package por

import (
	"context"
	"testing"
	"time"

	"bisonai.com/miko/node/pkg/fetcher"
	"github.com/stretchr/testify/assert"
)

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

	roundId, err := app.getRoundId(ctx, app.entries["peg-por"])
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
	info, err := app.getLastInfo(ctx, app.entries["peg-por"])
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
	err = app.execute(ctx, app.entries["peg-por"])
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

	value, err := fetcher.FetchSingle(ctx, app.entries["peg-por"].definition)
	if err != nil {
		t.Fatal(err)
	}

	roundId, err := app.getRoundId(ctx, app.entries["peg-por"])
	if err != nil {
		t.Fatal(err)
	}

	err = app.report(ctx, app.entries["peg-por"], value, roundId)
	if err != nil {
		t.Fatal(err)
	}
}

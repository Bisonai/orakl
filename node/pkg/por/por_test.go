package por

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNew(t *testing.T) {
	ctx := context.Background()
	app, err := New(ctx)
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, "PEG-POR", app.Name)
	assert.Equal(t, 60*time.Second, app.FetchInterval)
	assert.Equal(t, 60*time.Minute, app.SubmitInterval)
	assert.NotNil(t, app.KaiaHelper)
	assert.Equal(t, "0x58798D6Ca40480DF2FAd1b69939C3D29d91b60d3", app.ContractAddress)

	publicAddress, err := app.KaiaHelper.PublicAddressString()
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
	roundId, err := app.GetRoundID(ctx)
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
	info, err := app.GetLastInfo(ctx)
	if err != nil {
		t.Fatal(err)
	}
	updatedTime := time.Unix(info.UpdatedAt.Int64(), 0)
	assert.True(t, updatedTime.Before(time.Now()))

	answer := info.Answer.Int64()
	assert.Greater(t, answer, int64(0))
}

func TestFetch(t *testing.T) {
	ctx := context.Background()
	app, err := New(ctx)
	if err != nil {
		t.Fatal(err)
	}
	_, err = app.Fetch(ctx)
	if err != nil {
		t.Fatal(err)
	}
}

func TestExecute(t *testing.T) {
	ctx := context.Background()
	app, err := New(ctx)
	if err != nil {
		t.Fatal(err)
	}
	err = app.Execute(ctx)
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

	value, err := app.Fetch(ctx)
	if err != nil {
		t.Fatal(err)
	}

	roundId, err := app.GetRoundID(ctx)
	if err != nil {
		t.Fatal(err)
	}

	err = app.report(ctx, value, roundId)
	if err != nil {
		t.Fatal(err)
	}
}

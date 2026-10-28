package main

import (
	"bytes"
	"context"
	"path/filepath"
	"testing"
	"time"

	"bisonai.com/miko/node/pkg/chain/utils"
	"bisonai.com/miko/node/pkg/nodestate"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRunStateUsage(t *testing.T) {
	ctx := context.Background()
	for _, args := range [][]string{{}, {"bogus"}, {"list", "x"}, {"import", "x"}, {"import", "-nope"}} {
		var out, errOut bytes.Buffer
		err := runState(ctx, args, &out, &errOut)
		assert.ErrorContains(t, err, "usage: nodebin state", args)
	}
}

func TestRunStateListMissingFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "missing.db")
	t.Setenv(nodestate.EnvPath, path)
	var out, errOut bytes.Buffer
	assert.Error(t, runState(context.Background(), []string{"list"}, &out, &errOut))
}

func TestRunStateListNeverPrintsPk(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "node.db")
	t.Setenv(nodestate.EnvPath, path)

	s, err := nodestate.Open(ctx, path)
	require.NoError(t, err)
	addr := "0xabc"
	now := time.Now()
	require.NoError(t, s.Replace(ctx, nodestate.Snapshot{
		Keys: []utils.SignerKey{
			{Address: &addr, PK: "SECRET-ENCRYPTED-PK-1", State: "active", CreatedAt: now, UpdatedAt: now},
			{PK: "SECRET-ENCRYPTED-PK-2", State: "pending", CreatedAt: now, UpdatedAt: now},
		},
		Rounds: []nodestate.Round{{ConfigName: "BTC-USDT", Round: 9}},
	}))
	require.NoError(t, s.Close())

	var out, errOut bytes.Buffer
	require.NoError(t, runState(ctx, []string{"list"}, &out, &errOut))
	assert.Contains(t, out.String(), "0xabc\tactive")
	assert.Contains(t, out.String(), "-\tpending")
	assert.Contains(t, out.String(), "BTC-USDT\t9")
	assert.NotContains(t, out.String(), "SECRET")
	assert.NotContains(t, errOut.String(), "SECRET")
}

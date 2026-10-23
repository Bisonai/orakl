package main

import (
	"bytes"
	"context"
	"path/filepath"
	"strings"
	"testing"

	"bisonai.com/miko/node/pkg/boot/utils/peerstore"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRunPeers(t *testing.T) {
	ctx := context.Background()
	t.Setenv(peerstore.EnvPath, filepath.Join(t.TempDir(), "boot.db"))

	var out bytes.Buffer
	assert.Error(t, runPeers(ctx, nil, nil, &out))
	assert.Error(t, runPeers(ctx, []string{"list", "x"}, nil, &out))
	// list does not create a missing file
	assert.Error(t, runPeers(ctx, []string{"list"}, nil, &out))

	in := strings.NewReader("/ip4/127.0.0.1/tcp/10001/p2p/a\n/ip4/127.0.0.1/tcp/10002/p2p/b\n/ip4/127.0.0.1/tcp/10001/p2p/a\n")
	require.NoError(t, runPeers(ctx, []string{"import", "-"}, in, &out))
	assert.Equal(t, "imported 2 new peers\n", out.String())

	out.Reset()
	require.NoError(t, runPeers(ctx, []string{"list"}, nil, &out))
	assert.Equal(t, "1\t/ip4/127.0.0.1/tcp/10001/p2p/a\n2\t/ip4/127.0.0.1/tcp/10002/p2p/b\n", out.String())
}

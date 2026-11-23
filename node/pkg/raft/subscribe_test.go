package raft

import (
	"bytes"
	"context"
	"strings"
	"sync"
	"testing"
	"time"

)

// logs captures everything the package logs; TestMain points log.Logger at it.
var logs syncBuffer

type syncBuffer struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (b *syncBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.Write(p)
}

// since returns what was logged after offset (from len).
func (b *syncBuffer) since(offset int) string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.String()[offset:]
}

func (b *syncBuffer) len() int {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.Len()
}

// A stop or refresh cancels ctx while subscribe waits in sub.Next; that is not
// a topic failure and must not log an error (#2985).
func TestSubscribeCancelLogsNoError(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	node := newPublishTestNode(ctx, t)

	mark := logs.len()
	subCtx, stop := context.WithCancel(ctx)
	done := make(chan struct{})
	go func() {
		defer close(done)
		node.subscribe(subCtx)
	}()
	// let subscribe block in sub.Next
	time.Sleep(100 * time.Millisecond)
	stop()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("subscribe did not return after cancel")
	}

	out := logs.since(mark)
	for _, line := range strings.Split(out, "\n") {
		if strings.Contains(line, "failed to get message from topic") {
			t.Errorf("cancel logged a topic failure: %s", line)
		}
	}
	if !strings.Contains(out, "topic subscription stopped") {
		// a slow runner cancelled before subscribe reached sub.Next
		t.Skip("cancel did not reach sub.Next")
	}
}

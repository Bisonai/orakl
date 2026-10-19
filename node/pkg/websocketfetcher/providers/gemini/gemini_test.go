package gemini

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"bisonai.com/miko/node/pkg/websocketfetcher/common"
)

func TestCacheVolumesExitsOnCancel(t *testing.T) {
	var calls atomic.Int64
	called := make(chan struct{}, 1)
	orig := volumeFetcher
	volumeFetcher = func(map[string][]int32, *common.VolumeCacheMap) {
		calls.Add(1)
		select {
		case called <- struct{}{}:
		default:
		}
	}
	t.Cleanup(func() { volumeFetcher = orig })

	f := &GeminiFetcher{}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		f.CacheVolumes(ctx)
		close(done)
	}()

	select {
	case <-called:
	case <-time.After(2 * time.Second):
		t.Fatal("initial volume fetch not called")
	}
	cancel()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("CacheVolumes did not exit after ctx cancel")
	}
	if got := calls.Load(); got != 1 {
		t.Fatalf("expected 1 fetch call, got %d", got)
	}
}

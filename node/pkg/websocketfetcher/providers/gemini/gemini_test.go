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
	volumeFetcher = func(context.Context, map[string][]int32, *common.VolumeCacheMap) {
		calls.Add(1)
		select {
		case called <- struct{}{}:
		default:
		}
	}

	f := &GeminiFetcher{}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		f.CacheVolumes(ctx)
		close(done)
	}()
	// restore only once the poller has exited, so a leaked goroutine never races the write
	t.Cleanup(func() {
		cancel()
		select {
		case <-done:
			volumeFetcher = orig
		case <-time.After(2 * time.Second):
		}
	})

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

func TestFetchVolumesStopsOnCancel(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	feedMap := map[string][]int32{"btcusd": {1}, "ethusd": {2}}
	volumeCacheMap := &common.VolumeCacheMap{Map: make(map[int32]common.VolumeCache)}

	start := time.Now()
	fetchVolumes(ctx, feedMap, volumeCacheMap)
	if elapsed := time.Since(start); elapsed > 500*time.Millisecond {
		t.Fatalf("fetchVolumes did not return promptly on cancelled ctx: %v", elapsed)
	}
	if len(volumeCacheMap.Map) != 0 {
		t.Fatalf("expected no cached volumes, got %d", len(volumeCacheMap.Map))
	}
}

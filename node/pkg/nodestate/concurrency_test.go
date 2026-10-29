package nodestate

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// Node start: every aggregator reads its round at once while the round writer
// saves, some with contexts cancelled mid-query. Crashed (SIGSEGV in
// sqlite3_close_v2) when the pool opened and closed connections freely.
func TestConcurrentRoundAccess(t *testing.T) {
	s, _ := openTemp(t)
	ctx := context.Background()

	const configs = 250
	rounds := make(map[string]int32, configs)
	for i := 0; i < configs; i++ {
		rounds[fmt.Sprintf("C%d-USDT", i)] = int32(1000 + i)
	}
	require.NoError(t, s.SaveRounds(ctx, rounds))

	for iter := 0; iter < 20; iter++ {
		var wg sync.WaitGroup
		for i := 0; i < configs; i++ {
			wg.Add(1)
			go func(i int) {
				defer wg.Done()
				qctx, cancel := context.WithTimeout(ctx, time.Duration(i%7)*time.Microsecond)
				defer cancel()
				_, _, _ = s.LatestRound(qctx, fmt.Sprintf("C%d-USDT", i))
			}(i)
		}
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = s.SaveRounds(ctx, rounds)
		}()
		wg.Wait()
	}

	got, ok, err := s.LatestRound(ctx, "C0-USDT")
	require.NoError(t, err)
	require.True(t, ok)
	require.Equal(t, int32(1000), got)
}

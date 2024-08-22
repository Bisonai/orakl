package condition

import (
	"context"
	"time"

	errorsentinel "bisonai.com/miko/node/pkg/error"
)

// can be blocking infinitely if condition is not met, use with caution
func WaitForCondition(ctx context.Context, condition func() bool) {
	for {
		if condition() {
			return
		}

		select {
		case <-ctx.Done():
			return
		default:
			time.Sleep(500 * time.Millisecond)
		}
	}
}

func WaitForConditionWithTimeout(ctx context.Context, timeout time.Duration, condition func() bool) error {
	for {
		if condition() {
			return nil
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(timeout):
			return errorsentinel.ErrConditionTimedOut
		default:
			time.Sleep(500 * time.Millisecond)
		}
	}
}

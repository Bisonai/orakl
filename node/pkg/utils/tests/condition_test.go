package tests

import (
	"context"
	"testing"
	"time"

	"bisonai.com/miko/node/pkg/utils/condition"
	"github.com/stretchr/testify/assert"
)

func TestWaitForCondition_Success(t *testing.T) {
	conditionMet := false

	testCond := func() bool {
		return conditionMet
	}

	go func() {
		time.Sleep(100 * time.Millisecond)
		conditionMet = true
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	condition.WaitForCondition(ctx, testCond)

	assert.True(t, conditionMet)
}

func TestWaitForCondition_Timeout(t *testing.T) {
	testCond := func() bool {
		return false
	}

	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	condition.WaitForCondition(ctx, testCond)
}

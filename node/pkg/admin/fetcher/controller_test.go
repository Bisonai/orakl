package fetcher

import (
	"testing"

	"bisonai.com/miko/node/pkg/bus"
	errorSentinel "bisonai.com/miko/node/pkg/error"
	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
)

func TestResponseErrorStatus(t *testing.T) {
	inProgress := bus.MessageResponse{Args: map[string]any{"error": errorSentinel.ErrFetcherLifecycleInProgress.Error()}}
	assert.Equal(t, fiber.StatusConflict, responseErrorStatus(inProgress))

	other := bus.MessageResponse{Args: map[string]any{"error": "boom"}}
	assert.Equal(t, fiber.StatusInternalServerError, responseErrorStatus(other))
}

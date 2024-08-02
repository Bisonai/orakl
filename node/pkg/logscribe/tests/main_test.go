package test

import (
	"context"
	"encoding/json"
	"os"
	"testing"

	"bisonai.com/orakl/node/pkg/db"
	"bisonai.com/orakl/node/pkg/logscribe/api"
	"github.com/rs/zerolog"
)

func getInsertLogData() ([]api.LogInsertModel, error) {
	field1 := map[string]interface{}{
		"error":  "Service: Others, Code: InternalError, Message: Request status not OK",
		"Player": "Fetcher",
	}
	field2 := map[string]interface{}{
		"error":  "Failed to set reporters",
		"Player": "Reporter",
	}

	field1JsonData, err := json.Marshal(field1)
	if err != nil {
		return nil, err
	}
	field2JsonData, err := json.Marshal(field2)
	if err != nil {
		return nil, err
	}

	return []api.LogInsertModel{
		{
			Service:   "node",
			Timestamp: "2024-07-29 03:15:02+00",
			Level:     3,
			Message:   "error in requestFeed",
			Fields:    json.RawMessage(field1JsonData),
		},
		{
			Service:   "reporter",
			Timestamp: "2024-07-29 03:15:02+00",
			Level:     2,
			Message:   "error in requestFeed",
			Fields:    json.RawMessage(field2JsonData),
		},
	}, nil
}

func cleanup(ctx context.Context) func() {
	return func() {
		_ = db.QueryWithoutResult(ctx, "DELETE FROM logs", nil)
	}
}

func TestMain(m *testing.M) {
	zerolog.SetGlobalLevel(zerolog.InfoLevel)
	code := m.Run()
	db.ClosePool()
	os.Exit(code)
}

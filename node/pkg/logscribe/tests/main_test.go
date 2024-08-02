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

const insertLogDataCount = 1_000

func getInsertLogData() ([]api.LogInsertModel, error) {
	field := map[string]interface{}{
		"error":  "Service: Others, Code: InternalError, Message: Request status not OK",
		"Player": "Fetcher",
	}

	fieldJsonData, err := json.Marshal(field)
	if err != nil {
		return nil, err
	}

	data := make([]api.LogInsertModel, 0, insertLogDataCount)
	for i := 0; i < insertLogDataCount; i++ {
		data = append(data, api.LogInsertModel{
			Service:   "node",
			Timestamp: "2024-07-29 03:15:02+00",
			Level:     3,
			Message:   "error in requestFeed",
			Fields:    json.RawMessage(fieldJsonData),
		})
	}

	return data, nil
}

func cleanup(ctx context.Context) {
	_ = db.QueryWithoutResult(ctx, "DELETE FROM logs", nil)
}

func TestMain(m *testing.M) {
	cleanup(context.Background())
	zerolog.SetGlobalLevel(zerolog.InfoLevel)
	code := m.Run()
	db.ClosePool()
	os.Exit(code)
}

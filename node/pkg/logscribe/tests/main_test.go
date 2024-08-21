package test

import (
	"context"
	"encoding/json"
	"os"
	"testing"
	"time"

	"bisonai.com/miko/node/pkg/db"
	"bisonai.com/miko/node/pkg/logscribe/api"
	"github.com/rs/zerolog"
)

const TestService = "test"

func getInsertLogData(count int) ([]api.LogInsertModel, error) {
	field := map[string]interface{}{
		"error":  "Service: Others, Code: InternalError, Message: Request status not OK",
		"Player": "Fetcher",
	}

	fieldJsonData, err := json.Marshal(field)
	if err != nil {
		return nil, err
	}

	data := make([]api.LogInsertModel, 0, count)
	for i := 0; i < count; i++ {
		data = append(data, api.LogInsertModel{
			Service:   TestService,
			Timestamp: time.Now(),
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

package api

import (
	"encoding/json"
	"time"
)

type LogInsertModel struct {
	Service   string          `db:"service" json:"service"`
	Timestamp time.Time       `db:"timestamp" json:"timestamp"`
	Level     int             `db:"level" json:"level"`
	Message   string          `db:"message" json:"message"`
	Fields    json.RawMessage `db:"fields" json:"fields"`
}

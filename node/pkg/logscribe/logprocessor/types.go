package logprocessor

import (
	"encoding/json"
	"time"

	"github.com/google/go-github/github"
)

type LogInsertModel struct {
	Service   string          `db:"service" json:"service"`
	Timestamp time.Time       `db:"timestamp" json:"timestamp"`
	Level     int             `db:"level" json:"level"`
	Message   string          `db:"message" json:"message"`
	Fields    json.RawMessage `db:"fields" json:"fields"`
}

type LogProcessor struct {
	githubOwner  string
	githubRepo   string
	githubClient *github.Client
}

type LogInsertModelWithCount struct {
	LogInsertModel
	OccurrenceCount int `db:"occurrence_count" json:"occurrence_count"`
}

type LogsWithCount struct {
	count int
	log   LogInsertModel
}

type Count struct {
	Count int `db:"count"`
}

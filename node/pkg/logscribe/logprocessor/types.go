package logprocessor

import (
	"encoding/json"
	"time"

	"github.com/google/go-github/github"
	"github.com/robfig/cron/v3"
)

type LogInsertModel struct {
	Service   string          `db:"service" json:"service"`
	Timestamp time.Time       `db:"timestamp" json:"timestamp"`
	Level     int             `db:"level" json:"level"`
	Message   string          `db:"message" json:"message"`
	Fields    json.RawMessage `db:"fields" json:"fields"`
}

type LogProcessor struct {
	githubOwner          string
	githubRepo           string
	githubClient         *github.Client
	chain                string
	cron                 *cron.Cron
	bulkLogsCopyInterval time.Duration
}

type LogProcessorConfig struct {
	cron                 *cron.Cron
	bulkLogsCopyInterval time.Duration
}

type LogProcessingOption func(c *LogProcessorConfig)

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

type Service struct {
	Service string `db:"service"`
}

func WithBulkLogsCopyInterval(interval time.Duration) LogProcessingOption {
	return func(c *LogProcessorConfig) {
		c.bulkLogsCopyInterval = interval
	}
}

func WithCron(cron *cron.Cron) LogProcessingOption {
	return func(c *LogProcessorConfig) {
		c.cron = cron
	}
}

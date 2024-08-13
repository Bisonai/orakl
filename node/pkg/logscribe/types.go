package logscribe

import (
	"time"

	"bisonai.com/orakl/node/pkg/logscribe/api"
	"github.com/google/go-github/github"
	"github.com/robfig/cron/v3"
)

type LogInsertModel = api.LogInsertModel

type LogInsertModelWithCount struct {
	LogInsertModel
	OccurrenceCount int `db:"occurrence_count" json:"occurrence_count"`
}

type App struct {
	githubOwner          string
	githubRepo           string
	githubClient         *github.Client
	bulkLogsCopyInterval time.Duration
	cron                 *cron.Cron
}

type LogsWithCount struct {
	count int
	log   LogInsertModel
}

type Service struct {
	Service string `db:"service"`
}

type Count struct {
	Count int `db:"count"`
}

type AppOption func(c *AppConfig)

type AppConfig struct {
	bulkLogsCopyInterval time.Duration
	cron                 *cron.Cron
}

func WithBulkLogsCopyInterval(interval time.Duration) AppOption {
	return func(c *AppConfig) {
		c.bulkLogsCopyInterval = interval
	}
}

func WithCron(cron *cron.Cron) AppOption {
	return func(c *AppConfig) {
		c.cron = cron
	}
}

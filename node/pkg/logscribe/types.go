package logscribe

import (
	"time"

	"bisonai.com/orakl/node/pkg/logscribe/api"
	"github.com/google/go-github/github"
)

type LogInsertModelWithID struct {
	api.LogInsertModel
	ID int `db:"id" json:"id"`
}

type LogInsertModelWithIDWithCount struct {
	LogInsertModelWithID
	OccurrenceCount int `db:"occurrence_count" json:"occurrence_count"`
}

type App struct {
	githubOwner          string
	githubRepo           string
	githubClient         *github.Client
	bulkLogsCopyInterval time.Duration
}

type HashLogPairs struct {
	hash string
	logs []LogInsertModelWithID
}

type Count struct {
	Count int `db:"count"`
}

type AppOption func(c *AppConfig)

type AppConfig struct {
	bulkLogsCopyInterval time.Duration
}

func WithBulkLogsCopyInterval(interval time.Duration) AppOption {
	return func(c *AppConfig) {
		c.bulkLogsCopyInterval = interval
	}
}

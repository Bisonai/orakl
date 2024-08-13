package logscribe

import (
	"time"

	"bisonai.com/orakl/node/pkg/logscribe/api"
	"bisonai.com/orakl/node/pkg/logscribe/logprocessor"
	"github.com/robfig/cron/v3"
)

type LogInsertModel = api.LogInsertModel

type App struct {
	logProcessor         *logprocessor.LogProcessor
	bulkLogsCopyInterval time.Duration
	cron                 *cron.Cron
}

type Service struct {
	Service string `db:"service"`
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

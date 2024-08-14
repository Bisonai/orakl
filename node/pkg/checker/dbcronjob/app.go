package dbcronjob

import (
	"context"
	"errors"
	"fmt"
	"time"

	"bisonai.com/orakl/node/pkg/alert"
	"bisonai.com/orakl/node/pkg/db"
	"bisonai.com/orakl/node/pkg/secrets"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog/log"
)

const (
	DefaultCheckInterval   = 6 * time.Hour
	GetLast6hrsCronjobRuns = "SELECT database, command, status, start_time FROM cron.job_run_details WHERE start_time >= NOW() - interval '6 hour' AND status = 'failed'"
)

type CronJobResult struct {
	Database  string    `json:"database"`
	Command   string    `json:"command"`
	Status    string    `json:"status"`
	StartTime time.Time `json:"start_time"`
}

func Start(ctx context.Context) error {
	cronDbConnectionUrl := secrets.GetSecret("CRON_DB_CONNECTION_URL")
	if cronDbConnectionUrl == "" {
		log.Error().Msg("Missing CRON_DB_CONNECTION_URL")
		return errors.New("missing CRON_DB_CONNECTION_URL")
	}

	cronDB, err := db.GetTransientPool(ctx, cronDbConnectionUrl)
	if err != nil {
		log.Error().Err(err).Msg("Error getting CRON DB connection")
		return err
	}
	defer cronDB.Close()

	ticker := time.NewTicker(DefaultCheckInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Info().Msg("context cancelled, shutting down")
			return ctx.Err()
		case <-ticker.C:
			err := cronDBResult(ctx, cronDB)
			if err != nil {
				log.Error().Err(err).Msg("failed to get pgsql cron job run result")
			}
		}
	}
}

func cronDBResult(ctx context.Context, cronDB *pgxpool.Pool) error {
	result, err := db.QueryRowsTransient[CronJobResult](ctx, cronDB, GetLast6hrsCronjobRuns, nil)
	if err != nil {
		log.Error().Err(err).Msg("Error getting cron job results")
		return err
	}

	if len(result) == 0 {
		return nil
	}

	msg := ""

	for _, entry := range result {
		msg += fmt.Sprintf("FAILED PGSQL CRONJOB RUN\nDatabase: %s\nCommand: %s\nStatus: %s\nStart Time: %s\n\n", entry.Database, entry.Command, entry.Status, entry.StartTime)
	}

	if msg != "" {
		alert.SlackAlert(msg)
	}

	return nil
}

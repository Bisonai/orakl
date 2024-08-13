package dalstats

import (
	"context"
	"errors"
	"fmt"

	"bisonai.com/orakl/node/pkg/alert"
	"bisonai.com/orakl/node/pkg/db"
	"bisonai.com/orakl/node/pkg/secrets"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/robfig/cron/v3"
	"github.com/rs/zerolog/log"
)

const (
	GetAllValidKeys = "SELECT key, description FROM keys WHERE description NOT IN ('test', 'sentinel', 'orakl_reporter')"

	GetRestCallsPerKey              = "SELECT COUNT(1) FROM rest_calls WHERE api_key = @key AND timestamp >= NOW() - interval '7 day'"
	GetWebsocketConnectionsPerKey   = "SELECT COUNT(1) FROM websocket_connections WHERE api_key = @key AND timestamp >= NOW() - interval '7 day'"
	GetWebsocketSubscriptionsPerKey = "SELECT COUNT(1) FROM websocket_subscriptions WHERE id IN (SELECT id FROM websocket_connections WHERE api_key = @key AND timestamp >= NOW() - interval '7 day') AND timestamp >= NOW() - interval '7 day'"
)

type Key struct {
	key         string `db:"key"`
	description string `db:"description"`
}

type Count struct {
	count int `db:"count"`
}

// DAL Statistics report sent every friday
func Start(ctx context.Context) error {
	dalDBConnectionUrl := secrets.GetSecret("DAL_DB_CONNECTION_URL")
	if dalDBConnectionUrl == "" {
		log.Error().Msg("Missing DAL_DB_CONNECTION_URL")
		return errors.New("missing DAL_DB_CONNECTION_URL")
	}

	dalDB, err := db.GetTransientPool(ctx, dalDBConnectionUrl)
	if err != nil {
		log.Error().Err(err).Msg("Error getting DAL DB connection")
		return err
	}
	defer dalDB.Close()

	c := cron.New()
	_, err = c.AddFunc("0 1 * * 5", func() {
		err := dalDBStats(ctx, dalDB)
		if err != nil {
			log.Error().Err(err).Msg("Error running DAL DB stats")
		}
	})
	if err != nil {
		log.Error().Err(err).Msg("Error running DAL DB cron")
		return err
	}

	c.Start()
	<-ctx.Done()
	return nil
}

func dalDBStats(ctx context.Context, dalDB *pgxpool.Pool) error {
	keys, err := db.QueryRowsTransient[Key](ctx, dalDB, GetAllValidKeys, nil)
	if err != nil {
		log.Error().Err(err).Msg("Error getting keys")
		return err
	}
	if len(keys) == 0 {
		return nil
	}

	msg := ""

	for _, key := range keys {
		restCalls, err := db.QueryRowTransient[Count](ctx, dalDB, GetRestCallsPerKey, map[string]interface{}{"key": key.key})
		if err != nil {
			log.Error().Err(err).Msg("Error getting rest calls")
			return err
		}
		websocketConnections, err := db.QueryRowTransient[Count](ctx, dalDB, GetWebsocketConnectionsPerKey, map[string]interface{}{"key": key.key})
		if err != nil {
			log.Error().Err(err).Msg("Error getting websocket connections")
			return err
		}
		websocketSubscriptions, err := db.QueryRowTransient[Count](ctx, dalDB, GetWebsocketSubscriptionsPerKey, map[string]interface{}{"key": key.key})
		if err != nil {
			log.Error().Err(err).Msg("Error getting websocket subscriptions")
			return err
		}

		msg += fmt.Sprintf("(DAL 7 days calls) client: %s, rest calls: %v, websocket connections: %v, websocket subscriptions: %v\n", key.description, restCalls.count, websocketConnections.count, websocketSubscriptions.count)
	}

	if msg != "" {
		alert.SlackAlert(msg)
	}

	return nil
}

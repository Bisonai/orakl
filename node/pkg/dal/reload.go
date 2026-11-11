package dal

import (
	"context"

	"bisonai.com/miko/node/pkg/dal/collector"
	"bisonai.com/miko/node/pkg/dal/hub"
	"bisonai.com/miko/node/pkg/utils/configfetch"
	"bisonai.com/miko/node/pkg/utils/configreload"
	"github.com/rs/zerolog/log"
)

// startConfigReload re-reads the chain's config bundle while DAL runs (see
// configreload) and applies added/removed symbols to the collector and hub.
func startConfigReload(ctx context.Context, chain string, count int, c *collector.Collector, h *hub.Hub) {
	r := configreload.New(configfetch.FeedsPath(chain), count,
		func(config Config) string { return config.Name },
		func(ctx context.Context, configs []Config) error { return ApplyConfigs(ctx, c, h, configs) })
	go r.Run(ctx, configreload.Interval(), configreload.NotifyHUP())
}

// ApplyConfigs applies a reloaded bundle: added symbols become subscribable; removed ones stop being
// broadcast and are dropped from client subscriptions, then from the collector.
func ApplyConfigs(ctx context.Context, c *collector.Collector, h *hub.Hub, configs []Config) error {
	added, removed, err := c.UpdateConfigs(ctx, configs)
	if err != nil {
		return err
	}
	h.AddSymbols(added)
	h.RemoveSymbols(removed)
	c.RemoveSymbols(removed)

	if len(added) > 0 || len(removed) > 0 {
		addedNames := make([]string, 0, len(added))
		for name := range added {
			addedNames = append(addedNames, name)
		}
		log.Info().Strs("added", addedNames).Strs("removed", removed).Int("symbols", len(configs)).Msg("DAL config reloaded")
	}
	return nil
}

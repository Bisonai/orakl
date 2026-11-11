package reporter

import (
	"context"
	"maps"
	"sort"
	"time"

	"bisonai.com/miko/node/pkg/wss"
	"github.com/rs/zerolog/log"
)

// wsSwapGrace keeps the previous DAL websocket open this long after a reload
// opened one with the new subscriptions, so deviation data has no gap.
var wsSwapGrace = 10 * time.Second

// An added pair without websocket data after wsResubscribeDelay triggers a
// resubscribe, at most wsResubscribeAttempts times.
var (
	wsResubscribeDelay    = 5 * time.Minute
	wsResubscribeAttempts = 6
)

const defaultSubmitInterval = 5000

func submitInterval(c Config) int {
	if c.SubmitInterval != nil && *c.SubmitInterval > 0 {
		return *c.SubmitInterval
	}
	return defaultSubmitInterval
}

// pairIntervals maps each pair to its submit interval (ms).
func pairIntervals(configs []Config) map[string]int {
	out := make(map[string]int, len(configs))
	for _, c := range configs {
		out[c.Name] = submitInterval(c)
	}
	return out
}

// applyConfigs applies a reloaded bundle as a per-pair diff:
//   - an interval group that still exists gets its new pair list (used from its
//     next tick; a running job keeps its snapshot);
//   - a new interval gets a new reporter; an interval left without pairs is
//     stopped (its running job, if any, finishes; never cancelled mid-submit);
//   - the deviation reporter gets the full pair list;
//   - removed pairs are dropped from the latest-data maps;
//   - if the pair set changed, a new DAL websocket with the new subscriptions
//     replaces the old one.
//
// All reporters keep sharing the one chain helper (nonce manager). Unchanged
// pairs and groups are not touched.
func (a *App) applyConfigs(ctx context.Context, configs []Config) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	newPairs := pairIntervals(configs)
	if maps.Equal(newPairs, a.pairs) {
		return nil
	}

	var added, removed []string
	for name := range newPairs {
		if _, ok := a.pairs[name]; !ok {
			added = append(added, name)
		}
	}
	for name := range a.pairs {
		if _, ok := newPairs[name]; !ok {
			removed = append(removed, name)
		}
	}
	sort.Strings(added)
	sort.Strings(removed)

	// create everything that can fail first, so an error changes nothing and
	// the bundle is retried on the next reload
	var ws *wss.WebsocketHelper
	if len(added) > 0 || len(removed) > 0 {
		var err error
		ws, err = SetupDalWsHelper(ctx, configs, a.deps.dalWsEndpoint, a.deps.dalApiKey)
		if err != nil {
			return err
		}
	}
	grouped := groupConfigsBySubmitIntervals(configs)
	created := map[int]*Reporter{}
	for interval, groupConfigs := range grouped {
		if _, ok := a.groups[interval]; ok {
			continue
		}
		r, err := a.newReporter(ctx, interval, groupConfigs, ReportJob)
		if err != nil {
			return err
		}
		created[interval] = r
	}

	// shrink/stop first, then grow, so a pair moving between intervals is
	// dropped by its old group before the new one picks it up
	for interval, r := range a.groups {
		groupConfigs, ok := grouped[interval]
		if !ok {
			r.Stop()
			delete(a.groups, interval)
			continue
		}
		r.SetPairs(configNames(groupConfigs))
	}
	for interval, r := range created {
		a.groups[interval] = r
		go r.Run(ctx)
	}

	a.deviation.SetPairs(configNames(configs))

	for _, name := range removed {
		a.LatestDataMap.Delete(name)
		a.LatestSubmittedDataMap.Delete(name)
	}

	a.pairs = newPairs
	a.configs = configs
	a.rebuildReporterList()

	if ws != nil {
		a.swapWsHelper(ctx, ws)
		if len(removed) > 0 {
			// the old websocket may deliver removed pairs until it closes
			time.AfterFunc(wsSwapGrace+time.Second, func() { a.forgetRemoved(removed) })
		}
		if len(added) > 0 {
			a.scheduleResubscribe(ctx, added, 1)
		}
	}

	log.Info().Str("Player", "Reporter").Strs("added", added).Strs("removed", removed).
		Int("pairs", len(newPairs)).Int("reporters", len(a.Reporters)).Msg("reporter config reloaded")
	return nil
}

// swapWsHelper runs h and closes the previous DAL websocket after wsSwapGrace
// (both feed the same LatestDataMap). Caller holds a.mu.
func (a *App) swapWsHelper(ctx context.Context, h *wss.WebsocketHelper) {
	wsCtx, cancel := context.WithCancel(ctx)
	go h.Run(wsCtx, a.HandleWsMessage)
	if old := a.wsCancel; old != nil {
		time.AfterFunc(wsSwapGrace, old)
	}
	a.WsHelper, a.wsCancel = h, cancel
}

// forgetRemoved drops latest data of pairs that are (still) not configured.
func (a *App) forgetRemoved(pairs []string) {
	a.mu.Lock()
	defer a.mu.Unlock()
	for _, name := range pairs {
		if _, ok := a.pairs[name]; !ok {
			a.LatestDataMap.Delete(name)
		}
	}
}

// scheduleResubscribe re-opens the DAL websocket if an added pair still has
// no data after wsResubscribeDelay: DAL reloads on its own schedule and
// ignores a SUBSCRIBE for a symbol it does not know yet, and the websocket
// otherwise only reconnects every 12h.
func (a *App) scheduleResubscribe(ctx context.Context, added []string, attempt int) {
	time.AfterFunc(wsResubscribeDelay, func() {
		if ctx.Err() != nil {
			return
		}
		a.mu.Lock()
		defer a.mu.Unlock()
		var missing []string
		for _, name := range added {
			if _, configured := a.pairs[name]; !configured {
				continue
			}
			if _, ok := a.LatestDataMap.Load(name); !ok {
				missing = append(missing, name)
			}
		}
		if len(missing) == 0 {
			return
		}
		if attempt >= wsResubscribeAttempts {
			log.Warn().Str("Player", "Reporter").Strs("pairs", missing).Msg("no DAL websocket data for added pairs, giving up resubscribing")
			return
		}
		ws, err := SetupDalWsHelper(ctx, a.configs, a.deps.dalWsEndpoint, a.deps.dalApiKey)
		if err != nil {
			log.Error().Str("Player", "Reporter").Err(err).Msg("failed to create DAL websocket")
			return
		}
		log.Info().Str("Player", "Reporter").Strs("pairs", missing).Int("attempt", attempt).Msg("no DAL websocket data for added pairs, resubscribing")
		a.swapWsHelper(ctx, ws)
		a.scheduleResubscribe(ctx, missing, attempt+1)
	})
}

func configNames(configs []Config) []string {
	out := make([]string, len(configs))
	for i, c := range configs {
		out[i] = c.Name
	}
	return out
}

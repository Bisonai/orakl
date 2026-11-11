package reporter

import (
	"context"
	"maps"
	"sort"
	"time"

	"github.com/rs/zerolog/log"
)

// wsSwapGrace keeps the previous DAL websocket open this long after a reload
// opened one with the new subscriptions, so deviation data has no gap.
var wsSwapGrace = 10 * time.Second

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

	grouped := groupConfigsBySubmitIntervals(configs)

	// create new groups first so an error leaves everything as it was
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

	var added, removed []string
	for name := range newPairs {
		if _, ok := a.pairs[name]; !ok {
			added = append(added, name)
		}
	}
	for name := range a.pairs {
		if _, ok := newPairs[name]; !ok {
			removed = append(removed, name)
			a.LatestDataMap.Delete(name)
			a.LatestSubmittedDataMap.Delete(name)
		}
	}
	sort.Strings(added)
	sort.Strings(removed)

	if len(added) > 0 || len(removed) > 0 {
		a.swapWsHelper(ctx, configs)
	}

	a.pairs = newPairs
	a.rebuildReporterList()
	log.Info().Str("Player", "Reporter").Strs("added", added).Strs("removed", removed).
		Int("pairs", len(newPairs)).Int("reporters", len(a.Reporters)).Msg("reporter config reloaded")
	return nil
}

// swapWsHelper opens a DAL websocket subscribed to configs and closes the
// previous one after wsSwapGrace (both feed the same LatestDataMap). Caller
// holds a.mu. On error the previous websocket stays.
func (a *App) swapWsHelper(ctx context.Context, configs []Config) {
	h, err := SetupDalWsHelper(ctx, configs, a.deps.dalWsEndpoint, a.deps.dalApiKey)
	if err != nil {
		log.Error().Str("Player", "Reporter").Err(err).Msg("failed to create DAL websocket for reloaded config, keeping previous subscriptions")
		return
	}
	wsCtx, cancel := context.WithCancel(ctx)
	go h.Run(wsCtx, a.HandleWsMessage)
	if old := a.wsCancel; old != nil {
		time.AfterFunc(wsSwapGrace, old)
	}
	a.WsHelper, a.wsCancel = h, cancel
}

func configNames(configs []Config) []string {
	out := make([]string, len(configs))
	for i, c := range configs {
		out[i] = c.Name
	}
	return out
}

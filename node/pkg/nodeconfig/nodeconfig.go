// Package nodeconfig holds the node's configs and feeds in memory, loaded from
// the orakl-config bundle (sync at boot and on POST /api/v1/config/sync).
//
// IDs are process-local: a config gets an ID the first time its name is seen
// (a feed: its config + feed name) and keeps it for the process lifetime, so
// fetcher and aggregator maps keyed by ID stay consistent across syncs and
// refreshes. Topics, rounds and published data are keyed by name, never ID.
package nodeconfig

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"sort"
	"sync"

	"bisonai.com/miko/node/pkg/common/types"
	"bisonai.com/miko/node/pkg/utils/request"
)

// defaults for intervals missing from the bundle (former Postgres column defaults)
const (
	DefaultFetchInterval     = 2000
	DefaultAggregateInterval = 5000
	DefaultSubmitInterval    = 15000
)

type Feed = types.Feed

type Config struct {
	ID                   int32   `json:"id"`
	Name                 string  `json:"name"`
	FetchInterval        int32   `json:"fetchInterval"`
	AggregateInterval    int32   `json:"aggregateInterval"`
	SubmitInterval       int32   `json:"submitInterval"`
	Decimals             *int    `json:"decimals"`
	FeedDataFreshness    *int    `json:"feedDataFreshness"`
	MultiplyBy           *string `json:"multiplyBy"`
	MultiplyByReciprocal bool    `json:"multiplyByReciprocal"`
}

// Entry is one config of the orakl-config bundle.
type Entry struct {
	Name                 string      `json:"name"`
	FetchInterval        *int32      `json:"fetchInterval"`
	AggregateInterval    *int32      `json:"aggregateInterval"`
	SubmitInterval       *int32      `json:"submitInterval"`
	Decimals             *int        `json:"decimals"`
	FeedDataFreshness    *int        `json:"feedDataFreshness"`
	MultiplyBy           *string     `json:"multiplyBy"`
	MultiplyByReciprocal bool        `json:"multiplyByReciprocal"`
	Feeds                []FeedEntry `json:"feeds"`
}

type FeedEntry struct {
	Name       string          `json:"name"`
	Definition json.RawMessage `json:"definition"`
}

type feedKey struct {
	config string
	feed   string
}

// storedFeed caches the definition's "type", parsed once in Set.
type storedFeed struct {
	Feed
	typ     string
	hasType bool
}

// ConfigFeeds is a config with its feeds, from one consistent snapshot.
type ConfigFeeds struct {
	Config
	Feeds     []Feed // all feeds
	HTTPFeeds []Feed // feeds without definition "type" (plain HTTP requests)
}

type Store struct {
	mu      sync.RWMutex
	configs []Config     // ordered by ID
	feeds   []storedFeed // ordered by ID

	configIDs    map[string]int32
	feedIDs      map[feedKey]int32
	nextConfigID int32
	nextFeedID   int32
}

func New() *Store {
	return &Store{
		configIDs: map[string]int32{},
		feedIDs:   map[feedKey]int32{},
	}
}

var defaultStore = New()

// Default is the process-wide store used by nodebin.
func Default() *Store {
	return defaultStore
}

// URL is the orakl-config bundle for $CHAIN (default kairos).
func URL() string {
	chain := os.Getenv("CHAIN")
	if chain == "" {
		chain = "kairos"
	}
	return fmt.Sprintf("https://config.orakl.network/%s_feeds.json", chain)
}

// Sync loads the bundle from URL() and replaces the store content. On error
// the previous content is kept.
func (s *Store) Sync(ctx context.Context) error {
	entries, err := request.Request[[]Entry](request.WithEndpoint(URL()))
	if err != nil {
		return fmt.Errorf("load config bundle %s: %w", URL(), err)
	}
	return s.Set(entries)
}

// Set validates entries and replaces the store content. Running fetchers and
// aggregators pick it up on their next refresh.
func (s *Store) Set(entries []Entry) error {
	if len(entries) == 0 {
		return errors.New("config bundle is empty")
	}
	seenConfigs := make(map[string]bool, len(entries))
	for _, e := range entries {
		if e.Name == "" {
			return errors.New("config without name")
		}
		if seenConfigs[e.Name] {
			return fmt.Errorf("duplicate config %s", e.Name)
		}
		seenConfigs[e.Name] = true
		seenFeeds := make(map[string]bool, len(e.Feeds))
		for _, f := range e.Feeds {
			if f.Name == "" {
				return fmt.Errorf("config %s: feed without name", e.Name)
			}
			if seenFeeds[f.Name] {
				return fmt.Errorf("config %s: duplicate feed %s", e.Name, f.Name)
			}
			seenFeeds[f.Name] = true
			if !json.Valid(f.Definition) {
				return fmt.Errorf("config %s: feed %s: invalid definition", e.Name, f.Name)
			}
		}
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	configs := make([]Config, 0, len(entries))
	var feeds []storedFeed
	for _, e := range entries {
		id, ok := s.configIDs[e.Name]
		if !ok {
			s.nextConfigID++
			id = s.nextConfigID
			s.configIDs[e.Name] = id
		}
		configs = append(configs, Config{
			ID:                   id,
			Name:                 e.Name,
			FetchInterval:        orDefault(e.FetchInterval, DefaultFetchInterval),
			AggregateInterval:    orDefault(e.AggregateInterval, DefaultAggregateInterval),
			SubmitInterval:       orDefault(e.SubmitInterval, DefaultSubmitInterval),
			Decimals:             e.Decimals,
			FeedDataFreshness:    e.FeedDataFreshness,
			MultiplyBy:           e.MultiplyBy,
			MultiplyByReciprocal: e.MultiplyByReciprocal,
		})
		for _, f := range e.Feeds {
			key := feedKey{config: e.Name, feed: f.Name}
			feedID, ok := s.feedIDs[key]
			if !ok {
				s.nextFeedID++
				feedID = s.nextFeedID
				s.feedIDs[key] = feedID
			}
			definition := make(json.RawMessage, len(f.Definition))
			copy(definition, f.Definition)
			feed := Feed{ID: feedID, Name: f.Name, Definition: definition, ConfigID: id}
			typ, hasType := FeedType(feed)
			feeds = append(feeds, storedFeed{Feed: feed, typ: typ, hasType: hasType})
		}
	}
	sort.Slice(configs, func(i, j int) bool { return configs[i].ID < configs[j].ID })
	sort.Slice(feeds, func(i, j int) bool { return feeds[i].ID < feeds[j].ID })
	s.configs = configs
	s.feeds = feeds
	return nil
}

// Configs returns a copy of all configs, ordered by ID.
func (s *Store) Configs() []Config {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]Config, len(s.configs))
	copy(out, s.configs)
	return out
}

// ConfigsWithFeeds returns all configs (ordered by ID) with their feeds, read
// under one lock so a concurrent Set never mixes two bundles.
func (s *Store) ConfigsWithFeeds() []ConfigFeeds {
	s.mu.RLock()
	defer s.mu.RUnlock()
	byConfig := make(map[int32]*ConfigFeeds, len(s.configs))
	out := make([]ConfigFeeds, len(s.configs))
	for i, c := range s.configs {
		out[i].Config = c
		byConfig[c.ID] = &out[i]
	}
	for _, f := range s.feeds {
		cf, ok := byConfig[f.ConfigID]
		if !ok {
			continue
		}
		cf.Feeds = append(cf.Feeds, f.Feed)
		if !f.hasType {
			cf.HTTPFeeds = append(cf.HTTPFeeds, f.Feed)
		}
	}
	return out
}

// Feeds returns the feeds of a config, ordered by ID.
func (s *Store) Feeds(configID int32) []Feed {
	return s.filterFeeds(func(f storedFeed) bool { return f.ConfigID == configID })
}

// FeedsOfType returns all feeds whose definition "type" equals typ.
func (s *Store) FeedsOfType(typ string) []Feed {
	return s.filterFeeds(func(f storedFeed) bool { return f.typ == typ })
}

func (s *Store) filterFeeds(keep func(storedFeed) bool) []Feed {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var out []Feed
	for _, f := range s.feeds {
		if keep(f) {
			out = append(out, f.Feed)
		}
	}
	return out
}

// FeedType returns the definition's "type" string; hasType reports whether
// the key is present at all (a null or non-string type counts as present).
func FeedType(f Feed) (typ string, hasType bool) {
	var def map[string]json.RawMessage
	if err := json.Unmarshal(f.Definition, &def); err != nil {
		return "", false
	}
	raw, ok := def["type"]
	if !ok {
		return "", false
	}
	_ = json.Unmarshal(raw, &typ)
	return typ, true
}

func orDefault(v *int32, def int32) int32 {
	if v == nil || *v == 0 {
		return def
	}
	return *v
}

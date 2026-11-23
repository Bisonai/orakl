package apiv2

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"bisonai.com/miko/node/pkg/dal/collector"
	"github.com/rs/zerolog/log"
)

// historyRanges are the accepted values of the range parameter.
var historyRanges = map[string]time.Duration{
	"24h": 24 * time.Hour,
	"7d":  7 * 24 * time.Hour,
}

const defaultHistoryRange = "24h"

// maxHistoryPoints bounds the points one request may ask for (symbols x range/step),
// e.g. /history/all?range=7d needs step=1h. A var so tests can lower it.
var maxHistoryPoints int64 = 100_000

func (s *ServerV2) AllHistoryHandler(w http.ResponseWriter, r *http.Request) {
	s.serveHistory(w, r, s.collector.Symbols())
}

func (s *ServerV2) HistoryHandler(w http.ResponseWriter, r *http.Request) {
	symbols, err := s.parseHistorySymbols(r.PathValue("symbols"))
	if err != nil {
		writeText(w, http.StatusBadRequest, err.Error())
		return
	}
	s.serveHistory(w, r, symbols)
}

func (s *ServerV2) serveHistory(w http.ResponseWriter, r *http.Request, symbols []string) {
	if s.history == nil {
		writeText(w, http.StatusServiceUnavailable, "history disabled")
		return
	}
	window, step, err := parseHistoryParams(r.URL.Query(), s.history.Interval(), s.history.Retention())
	if err != nil {
		writeText(w, http.StatusBadRequest, err.Error())
		return
	}

	if points := int64(len(symbols)) * int64(window/step); points > maxHistoryPoints {
		writeText(w, http.StatusBadRequest, fmt.Sprintf("too many points (%d symbols x %d steps), use a larger step or fewer symbols", len(symbols), window/step))
		return
	}

	result, err := s.history.Query(r.Context(), symbols, time.Now().Add(-window), step)
	if err != nil {
		log.Error().Err(err).Msg("failed to query history")
		writeText(w, http.StatusInternalServerError, "failed to query history")
		return
	}
	for symbol, series := range result {
		// decimals as the latest-data endpoints serve them, for symbols without history in range
		if series.Decimals == "" {
			decimals, ok := s.collector.Decimals(symbol)
			if !ok { // removed by a reload during the request
				decimals = collector.DefaultDecimals
			}
			series.Decimals = decimals
			result[symbol] = series
		}
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(result); err != nil {
		log.Error().Err(err).Msg("failed to encode response")
	}
}

// parseHistorySymbols parses a comma-separated symbol list like
// /latest-data-feeds/{symbols}; unknown symbols are an error.
func (s *ServerV2) parseHistorySymbols(raw string) ([]string, error) {
	var symbols []string
	seen := map[string]bool{}
	for _, symbol := range strings.Split(strings.ReplaceAll(raw, " ", ""), ",") {
		if symbol == "" {
			continue
		}
		if !strings.Contains(symbol, "-") {
			return nil, fmt.Errorf("wrong symbol format: %s, symbol should be in {BASE}-{QUOTE} format", symbol)
		}
		if !strings.Contains(symbol, "test") {
			symbol = strings.ToUpper(symbol)
		}
		if _, ok := s.collector.Decimals(symbol); !ok {
			return nil, fmt.Errorf("unknown symbol: %s", symbol)
		}
		if !seen[symbol] {
			seen[symbol] = true
			symbols = append(symbols, symbol)
		}
	}
	if len(symbols) == 0 {
		return nil, fmt.Errorf("invalid symbol: empty symbol")
	}
	return symbols, nil
}

// parseHistoryParams validates range (24h or 7d, default 24h, within
// retention) and step (a multiple of interval, default interval, at most range).
func parseHistoryParams(q url.Values, interval, retention time.Duration) (time.Duration, time.Duration, error) {
	rangeStr := q.Get("range")
	if rangeStr == "" {
		rangeStr = defaultHistoryRange
	}
	window, ok := historyRanges[rangeStr]
	if !ok {
		return 0, 0, fmt.Errorf("invalid range %q: use 24h or 7d", rangeStr)
	}
	if window > retention {
		return 0, 0, fmt.Errorf("range %s exceeds history retention %s", rangeStr, retention)
	}

	step := interval
	if stepStr := q.Get("step"); stepStr != "" {
		parsed, err := time.ParseDuration(stepStr)
		if err != nil {
			return 0, 0, fmt.Errorf("invalid step %q", stepStr)
		}
		step = parsed
	}
	if step <= 0 || step%interval != 0 {
		return 0, 0, fmt.Errorf("step %s must be a positive multiple of %s", step, interval)
	}
	if step > window {
		return 0, 0, fmt.Errorf("step %s exceeds range %s", step, rangeStr)
	}
	return window, step, nil
}

func writeText(w http.ResponseWriter, status int, body string) {
	w.WriteHeader(status)
	if _, err := w.Write([]byte(body)); err != nil {
		log.Error().Err(err).Msg("failed to write response")
	}
}

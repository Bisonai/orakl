package price

import "strings"

type baseAndQuote struct {
	base, quote string
}

func (b *baseAndQuote) toBinanceSymbol() string {
	return strings.ToUpper(b.base + b.quote)
}

type dalResponse struct {
	Symbol, Value, Decimals string
}

// /api/v3/ticker/bookTicker element
type binanceResponse struct {
	Symbol   string `json:"symbol"`
	BidPrice string `json:"bidPrice"`
	AskPrice string `json:"askPrice"`
}

type App struct {
	dalApiKey          string
	trackingPairs      map[baseAndQuote]struct{}
	priceDiffThreshold float64
}

type Option func(*App)

func WithDalApiKey(apiKey string) Option {
	return func(a *App) {
		a.dalApiKey = apiKey
	}
}

func WithTrackingPairs(pairs []string) Option {
	return func(a *App) {
		for _, p := range pairs {
			splitted := strings.Split(p, "-")
			if len(splitted) != 2 {
				continue
			}
			a.trackingPairs[baseAndQuote{base: splitted[0], quote: splitted[1]}] = struct{}{}
		}
	}
}

func WithpriceDiffThreshold(threshold float64) Option {
	return func(a *App) {
		if threshold == 0 {
			return
		}
		a.priceDiffThreshold = threshold
	}
}

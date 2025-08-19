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

type binanceResponse struct {
	Symbol, Price string
}

type pythFeedResponse struct {
	Id         string
	Attributes pythFeedAttribute
}

type pythFeedAttribute struct {
	AssetType     string `json:"asset_type"`
	Base          string
	QuoteCurrency string `json:"quote_currency"`
}

type App struct {
	slackUrl                 string
	dalApiKey, slackEndpoint string
	trackingPairs            map[baseAndQuote]struct{}
	priceDiffThreshold       float64
}

type Option func(*App)

func WithDalApiKey(apiKey string) Option {
	return func(a *App) {
		a.dalApiKey = apiKey
	}
}

func WithSlackEndpoint(endpoint string) Option {
	return func(a *App) {
		a.slackEndpoint = endpoint
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

func WithSlackUrl(url string) Option {
	return func(a *App) {
		a.slackUrl = url
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

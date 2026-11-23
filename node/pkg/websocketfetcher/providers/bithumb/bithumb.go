package bithumb

import (
	"context"

	"bisonai.com/miko/node/pkg/websocketfetcher/common"
	"bisonai.com/miko/node/pkg/websocketfetcher/providers/upbit"
)

// URLV2 is bithumb's upbit-compatible websocket: same SIMPLE format for
// ticker (last trade, 24h volume, trade time) and orderbook messages. Unlike
// the old pubwss endpoint (URL) it streams top-of-book snapshots, which mid
// prices need.
const URLV2 = "wss://ws-api.bithumb.com/websocket/v1"

func New(ctx context.Context, opts ...common.FetcherOption) (common.FetcherInterface, error) {
	return upbit.NewCompatible(ctx, URLV2, "Bithumb", opts...)
}

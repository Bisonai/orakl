package wss

import "time"

// SetSubscribeDelay shortens the pause after dial and after each subscription
// for tests in package wss_test.
func SetSubscribeDelay(ws *WebsocketHelper, d time.Duration) {
	ws.subscribeDelay = d
}

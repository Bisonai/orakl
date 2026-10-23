package apiv2

import (
	"context"
	"net/http"
	"strconv"
	"strings"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// Values of the `key` label. Authenticated requests are labelled with the
// key's description from the key store, never with the raw key.
const (
	// UnknownKeyLabel labels requests rejected for a missing or invalid key.
	UnknownKeyLabel = "unknown"
	// KeyStoreErrorLabel labels requests rejected because the key store failed.
	KeyStoreErrorLabel = "error"
	// UnnamedKeyLabel labels valid keys that have an empty description.
	UnnamedKeyLabel = "unnamed"
	// OtherEndpointLabel labels requests that match no route.
	OtherEndpointLabel = "other"
)

var (
	wsActiveConnections = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "dal_websocket_active_connections",
		Help: "Current number of active WebSocket connections",
	}, []string{"key"})
	wsConnectionsTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "dal_websocket_connections_total",
		Help: "Total number of WebSocket connections",
	}, []string{"key"})
	wsSubscriptionsTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "dal_websocket_subscriptions_total",
		Help: "Total number of new WebSocket subscriptions to known symbols",
	}, []string{"key"})
	restRequestsTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "dal_rest_requests_total",
		Help: "Total number of REST API requests",
	}, []string{"key", "endpoint", "status"})
)

type keyLabelCtxKey struct{}

func withKeyLabel(ctx context.Context, label string) context.Context {
	return context.WithValue(ctx, keyLabelCtxKey{}, label)
}

func keyLabelFrom(ctx context.Context) string {
	if label, ok := ctx.Value(keyLabelCtxKey{}).(string); ok {
		return label
	}
	return UnknownKeyLabel
}

// keyLabel maps a key description to the `key` metric label. Descriptions
// equal to a reserved label, or already starting with the escape prefix, are
// prefixed, so distinct descriptions never share a label.
func keyLabel(description string) string {
	switch description {
	case "":
		return UnnamedKeyLabel
	case UnknownKeyLabel, KeyStoreErrorLabel, UnnamedKeyLabel:
		return keyLabelEscape + description
	}
	if strings.HasPrefix(description, keyLabelEscape) {
		return keyLabelEscape + description
	}
	return description
}

const keyLabelEscape = "key_"

// endpointLabel maps a ServeMux pattern ("GET /latest-data-feeds/{symbols}")
// to its route template ("/latest-data-feeds/{symbols}") so the label stays
// bounded regardless of the raw request path. No match, or only the "/"
// catch-all (health check, never metered itself), maps to "other".
func endpointLabel(pattern string) string {
	if pattern == "" || pattern == "/" {
		return OtherEndpointLabel
	}
	if i := strings.IndexByte(pattern, ' '); i >= 0 {
		pattern = pattern[i+1:]
	}
	return pattern
}

func observeREST(key, endpoint string, status int) {
	restRequestsTotal.WithLabelValues(key, endpoint, strconv.Itoa(status)).Inc()
}

// statusRecorder captures the response status code (200 unless set).
type statusRecorder struct {
	http.ResponseWriter
	status      int
	wroteHeader bool
}

func newStatusRecorder(w http.ResponseWriter) *statusRecorder {
	return &statusRecorder{ResponseWriter: w, status: http.StatusOK}
}

func (r *statusRecorder) WriteHeader(status int) {
	if !r.wroteHeader {
		r.status = status
		r.wroteHeader = true
	}
	r.ResponseWriter.WriteHeader(status)
}

func (r *statusRecorder) Write(b []byte) (int, error) {
	r.wroteHeader = true
	return r.ResponseWriter.Write(b)
}

func (r *statusRecorder) Unwrap() http.ResponseWriter {
	return r.ResponseWriter
}

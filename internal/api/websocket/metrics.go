package websocket

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// WebSocket connection gauge
	wsConnections = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "explorer_websocket_connections",
		Help: "Number of active WebSocket connections",
	})

	// WebSocket messages sent counter
	wsMessagesSent = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "explorer_websocket_messages_sent_total",
		Help: "Total number of WebSocket messages sent by type",
	}, []string{"channel"})

	// WebSocket errors counter
	wsErrors = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "explorer_websocket_errors_total",
		Help: "Total number of WebSocket errors by type",
	}, []string{"error_type"})
)

// UpdateConnectionMetrics updates the connection count gauge
func UpdateConnectionMetrics(count int) {
	wsConnections.Set(float64(count))
}

// IncrementMessageMetrics increments the message sent counter
func IncrementMessageMetrics(channel string, count int) {
	wsMessagesSent.WithLabelValues(channel).Add(float64(count))
}

// IncrementErrorMetrics increments the error counter
func IncrementErrorMetrics(errorType string) {
	wsErrors.WithLabelValues(errorType).Inc()
}

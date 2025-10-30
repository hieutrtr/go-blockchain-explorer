package util

import (
	"fmt"
	"log/slog"
	"net/http"
	"os"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	// BlocksIndexed tracks total number of blocks successfully indexed
	BlocksIndexed prometheus.Counter

	// IndexLagBlocks tracks number of blocks behind the network head
	IndexLagBlocks prometheus.Gauge

	// IndexLagSeconds tracks time lag in seconds from network head
	IndexLagSeconds prometheus.Gauge

	// RPCErrors tracks total number of RPC errors by error type
	RPCErrors prometheus.CounterVec

	// BackfillDuration tracks time to backfill a batch of blocks
	BackfillDuration prometheus.Histogram

	logger *slog.Logger
)

// Init initializes all Prometheus metrics and starts the metrics server
func Init() error {
	// Initialize logger with JSON output
	logger = slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))

	logger.Info("initializing prometheus metrics")

	// Register BlocksIndexed counter
	BlocksIndexed = promauto.NewCounter(prometheus.CounterOpts{
		Name: "explorer_blocks_indexed_total",
		Help: "Total number of blocks indexed",
	})

	// Register IndexLagBlocks gauge
	IndexLagBlocks = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "explorer_index_lag_blocks",
		Help: "Number of blocks behind network head",
	})

	// Register IndexLagSeconds gauge
	IndexLagSeconds = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "explorer_index_lag_seconds",
		Help: "Time lag from network head in seconds",
	})

	// Register RPCErrors counter vec with error_type label
	RPCErrors = *promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "explorer_rpc_errors_total",
			Help: "Total number of RPC errors by type",
		},
		[]string{"error_type"},
	)

	// Register BackfillDuration histogram with custom buckets
	BackfillDuration = promauto.NewHistogram(prometheus.HistogramOpts{
		Name:    "explorer_backfill_duration_seconds",
		Help:    "Time to backfill blocks (seconds)",
		Buckets: []float64{0.1, 0.5, 1.0, 2.0, 5.0, 10.0},
	})

	logger.Info("prometheus metrics initialized successfully")
	return nil
}

// RecordBlockIndexed increments the blocks indexed counter
func RecordBlockIndexed() {
	if BlocksIndexed == nil {
		logger.Warn("BlocksIndexed metric not initialized")
		return
	}
	BlocksIndexed.Inc()
}

// SetIndexLagBlocks sets the index lag in blocks
func SetIndexLagBlocks(lag float64) {
	if IndexLagBlocks == nil {
		logger.Warn("IndexLagBlocks metric not initialized")
		return
	}
	IndexLagBlocks.Set(lag)
}

// SetIndexLagSeconds sets the index lag in seconds
func SetIndexLagSeconds(lag float64) {
	if IndexLagSeconds == nil {
		logger.Warn("IndexLagSeconds metric not initialized")
		return
	}
	IndexLagSeconds.Set(lag)
}

// RecordRPCError increments the RPC errors counter for a specific error type
// errorType should be one of: network, rate_limit, invalid_param, timeout, other
func RecordRPCError(errorType string) {
	// Validate error type to prevent high-cardinality label explosion
	switch errorType {
	case "network", "rate_limit", "invalid_param", "timeout", "other":
		RPCErrors.WithLabelValues(errorType).Inc()
	default:
		logger.Warn("unknown RPC error type", "error_type", errorType)
		RPCErrors.WithLabelValues("other").Inc()
	}
}

// RecordBackfillDuration records the duration of a backfill batch in seconds
func RecordBackfillDuration(seconds float64) {
	if BackfillDuration == nil {
		logger.Warn("BackfillDuration metric not initialized")
		return
	}
	if seconds < 0 {
		logger.Warn("invalid backfill duration", "seconds", seconds)
		return
	}
	BackfillDuration.Observe(seconds)
}

// GetMetricsPort returns the configured metrics port from environment
func GetMetricsPort() string {
	port := os.Getenv("METRICS_PORT")
	if port == "" {
		port = "9090"
	}
	return port
}

// GetMetricsEndpoint returns the configured metrics endpoint from environment
func GetMetricsEndpoint() string {
	endpoint := os.Getenv("METRICS_ENDPOINT")
	if endpoint == "" {
		endpoint = "/metrics"
	}
	return endpoint
}

// StartMetricsServer starts an HTTP server serving Prometheus metrics
// This is called from cmd/worker/main.go
func StartMetricsServer() error {
	port := GetMetricsPort()
	endpoint := GetMetricsEndpoint()

	http.Handle(endpoint, promhttp.Handler())

	addr := fmt.Sprintf(":%s", port)
	logger.Info("starting metrics server",
		"address", addr,
		"endpoint", endpoint,
	)

	// This will block, so call it in a goroutine from main
	if err := http.ListenAndServe(addr, nil); err != nil {
		logger.Error("metrics server error", "error", err.Error())
		return fmt.Errorf("metrics server error: %w", err)
	}

	return nil
}

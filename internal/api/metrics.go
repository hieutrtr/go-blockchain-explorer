package api

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// apiRequestsTotal counts total API requests by method, endpoint, and status
	apiRequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "explorer_api_requests_total",
			Help: "Total number of API requests",
		},
		[]string{"method", "endpoint", "status"},
	)

	// apiLatencyMs records API request latency in milliseconds
	apiLatencyMs = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "explorer_api_latency_ms",
			Help:    "API request latency in milliseconds",
			Buckets: []float64{10, 25, 50, 100, 150, 200, 500, 1000, 2000, 5000},
		},
		[]string{"method", "endpoint"},
	)
)

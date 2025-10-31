package api

import (
	"net/http"
	"strings"
	"time"

	"github.com/hieutt50/go-blockchain-explorer/internal/util"
)

// loggingMiddleware logs all HTTP requests with method, path, status code, and latency
func (s *Server) loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Wrap response writer to capture status code
		ww := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}

		// Call next handler
		next.ServeHTTP(ww, r)

		// Calculate latency
		latency := time.Since(start)

		// Log request
		util.Info("API request",
			"method", r.Method,
			"path", r.URL.Path,
			"status", ww.statusCode,
			"latency_ms", latency.Milliseconds(),
			"remote_addr", r.RemoteAddr,
		)
	})
}

// corsMiddleware adds CORS headers to responses
func (s *Server) corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Parse allowed origins
		origins := s.config.CORSOrigins
		if origins == "" {
			origins = "*"
		}

		// Set CORS headers
		w.Header().Set("Access-Control-Allow-Origin", origins)
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		w.Header().Set("Access-Control-Max-Age", "86400") // 24 hours

		// Handle preflight requests
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// metricsMiddleware records Prometheus metrics for each request
func (s *Server) metricsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Wrap response writer to capture status code
		ww := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}

		// Call next handler
		next.ServeHTTP(ww, r)

		// Calculate latency
		latency := time.Since(start).Milliseconds()

		// Normalize path for metrics (remove specific IDs to avoid high cardinality)
		path := normalizePath(r.URL.Path)

		// Record metrics
		apiRequestsTotal.WithLabelValues(r.Method, path, http.StatusText(ww.statusCode)).Inc()
		apiLatencyMs.WithLabelValues(r.Method, path).Observe(float64(latency))
	})
}

// responseWriter wraps http.ResponseWriter to capture the status code
type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

// WriteHeader captures the status code
func (rw *responseWriter) WriteHeader(statusCode int) {
	rw.statusCode = statusCode
	rw.ResponseWriter.WriteHeader(statusCode)
}

// normalizePath removes specific IDs from paths to reduce cardinality in metrics
// Examples: /v1/blocks/12345 -> /v1/blocks/{height}, /v1/txs/0xabc -> /v1/txs/{hash}
func normalizePath(path string) string {
	if strings.HasPrefix(path, "/v1/blocks/") && len(path) > len("/v1/blocks/") {
		return "/v1/blocks/{id}"
	}
	if strings.HasPrefix(path, "/v1/txs/") && len(path) > len("/v1/txs/") {
		return "/v1/txs/{hash}"
	}
	if strings.HasPrefix(path, "/v1/address/") {
		// /v1/address/{addr}/txs
		if strings.Contains(path, "/txs") {
			return "/v1/address/{addr}/txs"
		}
		return "/v1/address/{addr}"
	}
	return path
}

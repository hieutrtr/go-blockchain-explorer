package api

import (
	"context"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/hieutt50/go-blockchain-explorer/internal/api/websocket"
	"github.com/hieutt50/go-blockchain-explorer/internal/db"
)

// Server holds the API server dependencies
type Server struct {
	pool   *db.Pool
	config *Config
	hub    *websocket.Hub
}

// NewServer creates a new API server instance
func NewServer(pool *db.Pool, config *Config) *Server {
	return &Server{
		pool:   pool,
		config: config,
		hub:    nil, // Hub will be set later if WebSocket is enabled
	}
}

// NewServerWithHub creates a new API server instance with WebSocket hub
func NewServerWithHub(pool *db.Pool, config *Config, hub *websocket.Hub) *Server {
	return &Server{
		pool:   pool,
		config: config,
		hub:    hub,
	}
}

// StartHub starts the WebSocket hub if present
func (s *Server) StartHub(ctx context.Context) {
	if s.hub != nil {
		go s.hub.Run(ctx)
	}
}

// Router configures and returns the HTTP router with all middleware and routes
func (s *Server) Router() http.Handler {
	r := chi.NewRouter()

	// Middleware stack
	r.Use(middleware.RequestID)      // Inject request ID
	r.Use(middleware.RealIP)         // Get real client IP
	r.Use(middleware.Recoverer)      // Recover from panics
	r.Use(s.loggingMiddleware)       // Log all requests
	r.Use(s.corsMiddleware)          // CORS headers
	r.Use(s.metricsMiddleware)       // Prometheus metrics

	// API v1 routes
	r.Route("/v1", func(r chi.Router) {
		// Block endpoints
		r.Get("/blocks", s.handleListBlocks)
		r.Get("/blocks/{heightOrHash}", s.handleGetBlock)

		// Transaction endpoints
		r.Get("/txs/{hash}", s.handleGetTransaction)

		// Address endpoints
		r.Get("/address/{addr}/txs", s.handleGetAddressTransactions)

		// Logs endpoints
		r.Get("/logs", s.handleQueryLogs)

		// Stats endpoints
		r.Get("/stats/chain", s.handleChainStats)

		// WebSocket endpoint
		if s.hub != nil {
			wsConfig := websocket.LoadConfig()
			r.Get("/stream", websocket.HandleWebSocket(s.hub, wsConfig))
		}
	})

	// Health check endpoint (no /v1 prefix)
	r.Get("/health", s.handleHealth)

	// Prometheus metrics endpoint
	r.Handle("/metrics", promhttp.Handler())

	// Static file serving for frontend (Story 2.4 will add files)
	r.Handle("/*", http.FileServer(http.Dir("./web")))

	return r
}

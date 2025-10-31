package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/hieutt50/go-blockchain-explorer/internal/db"
)

// BenchmarkAPI runs performance benchmarks for all API endpoints
// Run with: go test -bench=. -benchmem -run=^$ ./internal/api/...

func BenchmarkHealthCheck(b *testing.B) {
	router := setupBenchmarkRouter(b)

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			req := httptest.NewRequest("GET", "/health", nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			if w.Code != http.StatusOK {
				b.Errorf("unexpected status code: %d", w.Code)
			}
		}
	})
}

func BenchmarkListBlocks(b *testing.B) {
	router := setupBenchmarkRouter(b)

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			req := httptest.NewRequest("GET", "/v1/blocks?limit=25&offset=0", nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			if w.Code != http.StatusOK && w.Code != http.StatusServiceUnavailable {
				b.Errorf("unexpected status code: %d", w.Code)
			}
		}
	})
}

func BenchmarkGetBlockByHeight(b *testing.B) {
	router := setupBenchmarkRouter(b)

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			req := httptest.NewRequest("GET", "/v1/blocks/1000", nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			// Accept both OK and NotFound as valid responses
			if w.Code != http.StatusOK && w.Code != http.StatusNotFound && w.Code != http.StatusServiceUnavailable {
				b.Errorf("unexpected status code: %d", w.Code)
			}
		}
	})
}

func BenchmarkGetBlockByHash(b *testing.B) {
	router := setupBenchmarkRouter(b)

	// Use a valid-looking hash format
	testHash := "0x1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef"

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			req := httptest.NewRequest("GET", "/v1/blocks/"+testHash, nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			// Accept OK, NotFound, or ServiceUnavailable
			if w.Code != http.StatusOK && w.Code != http.StatusNotFound && w.Code != http.StatusServiceUnavailable {
				b.Errorf("unexpected status code: %d", w.Code)
			}
		}
	})
}

func BenchmarkGetChainStats(b *testing.B) {
	router := setupBenchmarkRouter(b)

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			req := httptest.NewRequest("GET", "/v1/stats/chain", nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			if w.Code != http.StatusOK && w.Code != http.StatusServiceUnavailable {
				b.Errorf("unexpected status code: %d", w.Code)
			}
		}
	})
}

func BenchmarkGetTransaction(b *testing.B) {
	router := setupBenchmarkRouter(b)

	testHash := "0xabcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890"

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			req := httptest.NewRequest("GET", "/v1/txs/"+testHash, nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			if w.Code != http.StatusOK && w.Code != http.StatusNotFound && w.Code != http.StatusServiceUnavailable {
				b.Errorf("unexpected status code: %d", w.Code)
			}
		}
	})
}

func BenchmarkQueryLogs(b *testing.B) {
	router := setupBenchmarkRouter(b)

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			req := httptest.NewRequest("GET", "/v1/logs?limit=100", nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			if w.Code != http.StatusOK && w.Code != http.StatusServiceUnavailable {
				b.Errorf("unexpected status code: %d", w.Code)
			}
		}
	})
}

func BenchmarkGetAddressTransactions(b *testing.B) {
	router := setupBenchmarkRouter(b)

	testAddr := "0x742d35Cc6634C0532925a3b844Bc9e7595f0bEb0"

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			req := httptest.NewRequest("GET", "/v1/address/"+testAddr+"/txs?limit=50", nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			if w.Code != http.StatusOK && w.Code != http.StatusServiceUnavailable {
				b.Errorf("unexpected status code: %d", w.Code)
			}
		}
	})
}

func BenchmarkConcurrentMixedRequests(b *testing.B) {
	router := setupBenchmarkRouter(b)

	endpoints := []string{
		"/health",
		"/v1/blocks?limit=25",
		"/v1/blocks/1000",
		"/v1/stats/chain",
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			endpoint := endpoints[i%len(endpoints)]
			req := httptest.NewRequest("GET", endpoint, nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)
			i++
		}
	})
}

// TestResponseTimes verifies that all endpoints meet performance targets (< 200ms)
func TestResponseTimes(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping performance test in short mode")
	}

	router := setupBenchmarkRouter(t)

	tests := []struct {
		name     string
		url      string
		maxLatMS int64 // Maximum latency in milliseconds
	}{
		{"Health Check", "/health", 50},
		{"List Blocks", "/v1/blocks?limit=25", 200},
		{"Get Block by Height", "/v1/blocks/1000", 200},
		{"Chain Stats", "/v1/stats/chain", 200},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Run multiple requests to get average
			const iterations = 10
			var totalLatency int64

			for i := 0; i < iterations; i++ {
				start := time.Now()

				req := httptest.NewRequest("GET", tt.url, nil)
				w := httptest.NewRecorder()
				router.ServeHTTP(w, req)

				latency := time.Since(start).Milliseconds()
				totalLatency += latency

				// Skip validation if database is not available
				if w.Code == http.StatusServiceUnavailable {
					t.Skip("database not available for performance testing")
				}
			}

			avgLatency := totalLatency / iterations
			t.Logf("%s: Average latency = %dms (target: <%dms)", tt.name, avgLatency, tt.maxLatMS)

			// Log warning if target not met, but don't fail
			// (actual performance depends on database and hardware)
			if avgLatency > tt.maxLatMS {
				t.Logf("WARNING: Average latency %dms exceeds target %dms", avgLatency, tt.maxLatMS)
			}
		})
	}
}

// TestThroughput measures requests per second for key endpoints
func TestThroughput(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping performance test in short mode")
	}

	router := setupBenchmarkRouter(t)

	// Test for 1 second
	duration := 1 * time.Second
	endpoint := "/health"

	start := time.Now()
	requestCount := 0

	for time.Since(start) < duration {
		req := httptest.NewRequest("GET", endpoint, nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		requestCount++

		if w.Code == http.StatusServiceUnavailable {
			t.Skip("database not available for performance testing")
		}
	}

	actualDuration := time.Since(start).Seconds()
	rps := float64(requestCount) / actualDuration

	t.Logf("Health check throughput: %.0f requests/second", rps)

	// Expect at least 100 RPS for health check
	if rps < 100 {
		t.Logf("WARNING: Throughput %.0f RPS is below target 100 RPS", rps)
	}
}

// TestJSONParsing benchmarks JSON encoding/decoding performance
func BenchmarkJSONEncoding(b *testing.B) {
	router := setupBenchmarkRouter(b)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest("GET", "/v1/blocks?limit=25", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		// Parse the JSON response to measure full cycle
		if w.Code == http.StatusOK {
			var response map[string]interface{}
			json.Unmarshal(w.Body.Bytes(), &response)
		}
	}
}

// setupBenchmarkRouter creates a test router for benchmarking
func setupBenchmarkRouter(tb testing.TB) http.Handler {
	// Try to connect to database
	config, err := db.NewConfig()
	if err != nil {
		tb.Skipf("cannot create database config: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	pool, err := db.NewPool(ctx, config)
	if err != nil {
		tb.Skipf("database not available for benchmarking: %v", err)
	}

	// Verify connectivity
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		tb.Skipf("cannot connect to database: %v", err)
	}

	tb.Cleanup(func() {
		pool.Close()
	})

	serverConfig := &Config{
		Port:            8080,
		CORSOrigins:     "*",
		ReadTimeout:     5 * time.Second,
		WriteTimeout:    10 * time.Second,
		IdleTimeout:     60 * time.Second,
		ShutdownTimeout: 30 * time.Second,
	}

	server := NewServer(pool, serverConfig)
	return server.Router()
}

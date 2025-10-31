package api

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNormalizePath(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		expected string
	}{
		{
			name:     "block by height",
			path:     "/v1/blocks/12345",
			expected: "/v1/blocks/{id}",
		},
		{
			name:     "block by hash",
			path:     "/v1/blocks/0xabcd...",
			expected: "/v1/blocks/{id}",
		},
		{
			name:     "transaction by hash",
			path:     "/v1/txs/0x123abc...",
			expected: "/v1/txs/{hash}",
		},
		{
			name:     "address transactions",
			path:     "/v1/address/0xabc.../txs",
			expected: "/v1/address/{addr}/txs",
		},
		{
			name:     "blocks list",
			path:     "/v1/blocks",
			expected: "/v1/blocks",
		},
		{
			name:     "logs",
			path:     "/v1/logs",
			expected: "/v1/logs",
		},
		{
			name:     "health",
			path:     "/health",
			expected: "/health",
		},
		{
			name:     "metrics",
			path:     "/metrics",
			expected: "/metrics",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := normalizePath(tt.path)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestCORSMiddleware(t *testing.T) {
	tests := []struct {
		name                   string
		corsOrigins            string
		method                 string
		expectedOrigin         string
		expectedMethods        string
		expectedHeaders        string
		expectedStatus         int
		shouldCallNextHandler  bool
	}{
		{
			name:                  "regular request with wildcard origins",
			corsOrigins:           "*",
			method:                "GET",
			expectedOrigin:        "*",
			expectedMethods:       "GET, POST, PUT, DELETE, OPTIONS",
			expectedHeaders:       "Content-Type, Authorization",
			expectedStatus:        http.StatusOK,
			shouldCallNextHandler: true,
		},
		{
			name:                  "regular request with specific origin",
			corsOrigins:           "https://example.com",
			method:                "POST",
			expectedOrigin:        "https://example.com",
			expectedMethods:       "GET, POST, PUT, DELETE, OPTIONS",
			expectedHeaders:       "Content-Type, Authorization",
			expectedStatus:        http.StatusOK,
			shouldCallNextHandler: true,
		},
		{
			name:                  "preflight OPTIONS request",
			corsOrigins:           "*",
			method:                "OPTIONS",
			expectedOrigin:        "*",
			expectedMethods:       "GET, POST, PUT, DELETE, OPTIONS",
			expectedHeaders:       "Content-Type, Authorization",
			expectedStatus:        http.StatusNoContent,
			shouldCallNextHandler: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create test server
			config := &Config{CORSOrigins: tt.corsOrigins}
			server := &Server{config: config}

			// Track if next handler was called
			nextHandlerCalled := false
			nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				nextHandlerCalled = true
				w.WriteHeader(http.StatusOK)
			})

			// Wrap with CORS middleware
			handler := server.corsMiddleware(nextHandler)

			// Create test request
			req := httptest.NewRequest(tt.method, "/test", nil)
			w := httptest.NewRecorder()

			// Execute request
			handler.ServeHTTP(w, req)

			// Verify CORS headers
			assert.Equal(t, tt.expectedOrigin, w.Header().Get("Access-Control-Allow-Origin"))
			assert.Equal(t, tt.expectedMethods, w.Header().Get("Access-Control-Allow-Methods"))
			assert.Equal(t, tt.expectedHeaders, w.Header().Get("Access-Control-Allow-Headers"))

			// Verify status and next handler execution
			assert.Equal(t, tt.expectedStatus, w.Code)
			assert.Equal(t, tt.shouldCallNextHandler, nextHandlerCalled)
		})
	}
}

func TestResponseWriter(t *testing.T) {
	t.Run("captures status code", func(t *testing.T) {
		w := httptest.NewRecorder()
		rw := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}

		// Write custom status
		rw.WriteHeader(http.StatusBadRequest)

		assert.Equal(t, http.StatusBadRequest, rw.statusCode)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("default status is 200", func(t *testing.T) {
		w := httptest.NewRecorder()
		rw := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}

		assert.Equal(t, http.StatusOK, rw.statusCode)
	})
}

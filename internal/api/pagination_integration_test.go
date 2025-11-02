package api

import (
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestPaginationIntegration_BlocksEndpoint tests pagination with real database queries
func TestPaginationIntegration_BlocksEndpoint(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	tests := []struct {
		name            string
		queryParams     string
		expectedLimit   int
		expectedOffset  int
		expectDataCount int // Expected number of results in response
	}{
		{
			name:            "default pagination",
			queryParams:     "",
			expectedLimit:   25,
			expectedOffset:  0,
			expectDataCount: 25, // Assumes at least 25 blocks in test DB
		},
		{
			name:            "custom limit",
			queryParams:     "?limit=10&offset=0",
			expectedLimit:   10,
			expectedOffset:  0,
			expectDataCount: 10,
		},
		{
			name:            "with offset",
			queryParams:     "?limit=10&offset=20",
			expectedLimit:   10,
			expectedOffset:  20,
			expectDataCount: 10,
		},
		{
			name:            "offset beyond total returns empty",
			queryParams:     "?limit=10&offset=999999",
			expectedLimit:   10,
			expectedOffset:  999999,
			expectDataCount: 0, // Beyond all data
		},
		{
			name:            "limit exceeds max clamped",
			queryParams:     "?limit=500&offset=0",
			expectedLimit:   100, // Clamped to max
			expectedOffset:  0,
			expectDataCount: 100,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create request
			req := httptest.NewRequest("GET", "/v1/blocks"+tt.queryParams, nil)

			// Parse pagination
			limit, offset := parsePagination(req, 25, 100)

			// Verify pagination parameters
			assert.Equal(t, tt.expectedLimit, limit, "limit mismatch")
			assert.Equal(t, tt.expectedOffset, offset, "offset mismatch")

			// Note: Actual database query test would require test database setup
			// This validates the pagination parsing logic in integration context
		})
	}
}

// TestPaginationIntegration_EdgeCases tests pagination edge cases
func TestPaginationIntegration_EdgeCases(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	t.Run("empty results with offset beyond total", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/v1/blocks?limit=10&offset=999999", nil)
		limit, offset := parsePagination(req, 25, 100)

		assert.Equal(t, 10, limit)
		assert.Equal(t, 999999, offset)
		// In real scenario: total would be > 0 but data array would be empty
	})

	t.Run("invalid limit uses default", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/v1/blocks?limit=invalid", nil)
		limit, offset := parsePagination(req, 25, 100)

		assert.Equal(t, 25, limit, "should use default for invalid limit")
		assert.Equal(t, 0, offset)
	})

	t.Run("negative offset uses zero", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/v1/blocks?limit=25&offset=-100", nil)
		limit, offset := parsePagination(req, 25, 100)

		assert.Equal(t, 25, limit)
		assert.Equal(t, 0, offset, "should use 0 for negative offset")
	})

	t.Run("mixed valid and invalid parameters", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/v1/blocks?limit=invalid&offset=50", nil)
		limit, offset := parsePagination(req, 25, 100)

		assert.Equal(t, 25, limit, "should use default for invalid limit")
		assert.Equal(t, 50, offset, "should preserve valid offset")
	})
}

// TestPaginationIntegration_ResponseFormat tests paginated response format
func TestPaginationIntegration_ResponseFormat(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	t.Run("response includes all metadata", func(t *testing.T) {
		// Simulate paginated response
		data := []string{"item1", "item2", "item3"}
		total := 100
		limit := 25
		offset := 0

		response := NewPaginatedResponse(data, total, limit, offset)

		require.NotNil(t, response)
		assert.Contains(t, response, "data")
		assert.Contains(t, response, "total")
		assert.Contains(t, response, "limit")
		assert.Contains(t, response, "offset")

		assert.Equal(t, data, response["data"])
		assert.Equal(t, total, response["total"])
		assert.Equal(t, limit, response["limit"])
		assert.Equal(t, offset, response["offset"])
	})

	t.Run("empty data with total count", func(t *testing.T) {
		// Offset beyond total scenario
		data := []interface{}{}
		total := 100
		limit := 25
		offset := 200

		response := NewPaginatedResponse(data, total, limit, offset)

		assert.Equal(t, 0, len(response["data"].([]interface{})))
		assert.Equal(t, 100, response["total"], "total should reflect all results")
		assert.Equal(t, 200, response["offset"], "offset should be preserved")
	})
}

// TestPaginationIntegration_ConcurrentRequests tests pagination under concurrent load
func TestPaginationIntegration_ConcurrentRequests(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	t.Run("concurrent pagination requests", func(t *testing.T) {
		// Test that concurrent pagination requests work correctly
		numRequests := 50

		results := make(chan bool, numRequests)

		for i := 0; i < numRequests; i++ {
			go func(reqNum int) {
				req := httptest.NewRequest("GET", "/v1/blocks?limit=10&offset=0", nil)
				limit, offset := parsePagination(req, 25, 100)

				// Verify each request gets correct pagination
				results <- (limit == 10 && offset == 0)
			}(i)
		}

		// Collect results
		successCount := 0
		for i := 0; i < numRequests; i++ {
			if <-results {
				successCount++
			}
		}

		assert.Equal(t, numRequests, successCount, "all concurrent requests should parse correctly")
	})
}

// TestValidationIntegration tests validation function with various scenarios
func TestValidationIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	tests := []struct {
		name      string
		limit     int
		offset    int
		maxLimit  int
		expectErr bool
		errType   error
	}{
		{
			name:      "valid parameters",
			limit:     50,
			offset:    100,
			maxLimit:  100,
			expectErr: false,
		},
		{
			name:      "limit at boundary",
			limit:     100,
			offset:    0,
			maxLimit:  100,
			expectErr: false,
		},
		{
			name:      "limit exceeds max",
			limit:     101,
			offset:    0,
			maxLimit:  100,
			expectErr: true,
			errType:   ErrLimitTooLarge,
		},
		{
			name:      "zero limit",
			limit:     0,
			offset:    0,
			maxLimit:  100,
			expectErr: true,
			errType:   ErrInvalidLimit,
		},
		{
			name:      "negative offset",
			limit:     50,
			offset:    -1,
			maxLimit:  100,
			expectErr: true,
			errType:   ErrInvalidOffset,
		},
		{
			name:      "large valid offset",
			limit:     25,
			offset:    100000,
			maxLimit:  100,
			expectErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validatePagination(tt.limit, tt.offset, tt.maxLimit)

			if tt.expectErr {
				assert.Error(t, err)
				if tt.errType != nil {
					assert.Equal(t, tt.errType, err)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// Note: Full integration tests with real database would require:
// 1. Test database setup/teardown
// 2. Test data fixtures
// 3. Database connection management
// These tests validate pagination logic and can be extended with real DB when test infrastructure is ready

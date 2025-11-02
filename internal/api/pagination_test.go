package api

import (
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParsePagination(t *testing.T) {
	tests := []struct {
		name          string
		queryParams   string
		defaultLimit  int
		maxLimit      int
		expectedLimit int
		expectedOffset int
	}{
		{
			name:           "no params - use defaults",
			queryParams:    "",
			defaultLimit:   25,
			maxLimit:       100,
			expectedLimit:  25,
			expectedOffset: 0,
		},
		{
			name:           "valid limit and offset",
			queryParams:    "?limit=50&offset=100",
			defaultLimit:   25,
			maxLimit:       100,
			expectedLimit:  50,
			expectedOffset: 100,
		},
		{
			name:           "limit exceeds max - clamp to max",
			queryParams:    "?limit=200",
			defaultLimit:   25,
			maxLimit:       100,
			expectedLimit:  100,
			expectedOffset: 0,
		},
		{
			name:           "invalid limit - use default",
			queryParams:    "?limit=invalid",
			defaultLimit:   25,
			maxLimit:       100,
			expectedLimit:  25,
			expectedOffset: 0,
		},
		{
			name:           "zero limit - use default",
			queryParams:    "?limit=0",
			defaultLimit:   25,
			maxLimit:       100,
			expectedLimit:  25,
			expectedOffset: 0,
		},
		{
			name:           "negative limit - use default",
			queryParams:    "?limit=-10",
			defaultLimit:   25,
			maxLimit:       100,
			expectedLimit:  25,
			expectedOffset: 0,
		},
		{
			name:           "invalid offset - use zero",
			queryParams:    "?offset=invalid",
			defaultLimit:   25,
			maxLimit:       100,
			expectedLimit:  25,
			expectedOffset: 0,
		},
		{
			name:           "negative offset - use zero",
			queryParams:    "?offset=-50",
			defaultLimit:   25,
			maxLimit:       100,
			expectedLimit:  25,
			expectedOffset: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/test"+tt.queryParams, nil)
			limit, offset := parsePagination(req, tt.defaultLimit, tt.maxLimit)

			assert.Equal(t, tt.expectedLimit, limit, "limit mismatch")
			assert.Equal(t, tt.expectedOffset, offset, "offset mismatch")
		})
	}
}

func TestValidatePagination(t *testing.T) {
	tests := []struct {
		name      string
		limit     int
		offset    int
		maxLimit  int
		expectErr bool
		errorType error
	}{
		{
			name:      "valid pagination",
			limit:     25,
			offset:    0,
			maxLimit:  100,
			expectErr: false,
		},
		{
			name:      "limit too large",
			limit:     200,
			offset:    0,
			maxLimit:  100,
			expectErr: true,
			errorType: ErrLimitTooLarge,
		},
		{
			name:      "zero limit",
			limit:     0,
			offset:    0,
			maxLimit:  100,
			expectErr: true,
			errorType: ErrInvalidLimit,
		},
		{
			name:      "negative offset",
			limit:     25,
			offset:    -1,
			maxLimit:  100,
			expectErr: true,
			errorType: ErrInvalidOffset,
		},
		{
			name:      "negative limit",
			limit:     -10,
			offset:    0,
			maxLimit:  100,
			expectErr: true,
			errorType: ErrInvalidLimit,
		},
		{
			name:      "limit equals max",
			limit:     100,
			offset:    0,
			maxLimit:  100,
			expectErr: false,
		},
		{
			name:      "limit exceeds max by 1",
			limit:     101,
			offset:    0,
			maxLimit:  100,
			expectErr: true,
			errorType: ErrLimitTooLarge,
		},
		{
			name:      "large valid offset",
			limit:     25,
			offset:    10000,
			maxLimit:  100,
			expectErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validatePagination(tt.limit, tt.offset, tt.maxLimit)
			if tt.expectErr {
				assert.Error(t, err)
				if tt.errorType != nil {
					assert.Equal(t, tt.errorType, err)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestNewPaginatedResponse(t *testing.T) {
	tests := []struct {
		name   string
		data   interface{}
		total  int
		limit  int
		offset int
	}{
		{
			name:   "with data",
			data:   []string{"item1", "item2"},
			total:  100,
			limit:  25,
			offset: 0,
		},
		{
			name:   "empty data",
			data:   []string{},
			total:  0,
			limit:  25,
			offset: 0,
		},
		{
			name:   "with offset",
			data:   []int{1, 2, 3},
			total:  50,
			limit:  10,
			offset: 20,
		},
		{
			name:   "nil data",
			data:   nil,
			total:  0,
			limit:  25,
			offset: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			response := NewPaginatedResponse(tt.data, tt.total, tt.limit, tt.offset)

			assert.NotNil(t, response)
			assert.Equal(t, tt.data, response["data"])
			assert.Equal(t, tt.total, response["total"])
			assert.Equal(t, tt.limit, response["limit"])
			assert.Equal(t, tt.offset, response["offset"])

			// Verify all expected keys are present
			assert.Contains(t, response, "data")
			assert.Contains(t, response, "total")
			assert.Contains(t, response, "limit")
			assert.Contains(t, response, "offset")
		})
	}
}

func TestValidationError(t *testing.T) {
	tests := []struct {
		name    string
		err     error
		wantMsg string
	}{
		{
			name:    "invalid limit error",
			err:     ErrInvalidLimit,
			wantMsg: "limit must be greater than 0",
		},
		{
			name:    "limit too large error",
			err:     ErrLimitTooLarge,
			wantMsg: "limit exceeds maximum allowed value",
		},
		{
			name:    "invalid offset error",
			err:     ErrInvalidOffset,
			wantMsg: "offset must be greater than or equal to 0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.wantMsg, tt.err.Error())
		})
	}
}

func TestParsePaginationEdgeCases(t *testing.T) {
	tests := []struct {
		name           string
		queryParams    string
		defaultLimit   int
		maxLimit       int
		expectedLimit  int
		expectedOffset int
	}{
		{
			name:           "very large limit clamped",
			queryParams:    "?limit=999999",
			defaultLimit:   25,
			maxLimit:       100,
			expectedLimit:  100,
			expectedOffset: 0,
		},
		{
			name:           "very large offset allowed",
			queryParams:    "?offset=999999",
			defaultLimit:   25,
			maxLimit:       100,
			expectedLimit:  25,
			expectedOffset: 999999,
		},
		{
			name:           "string 'abc' for limit",
			queryParams:    "?limit=abc",
			defaultLimit:   25,
			maxLimit:       100,
			expectedLimit:  25,
			expectedOffset: 0,
		},
		{
			name:           "string 'xyz' for offset",
			queryParams:    "?offset=xyz",
			defaultLimit:   25,
			maxLimit:       100,
			expectedLimit:  25,
			expectedOffset: 0,
		},
		{
			name:           "limit=1 (minimum valid)",
			queryParams:    "?limit=1",
			defaultLimit:   25,
			maxLimit:       100,
			expectedLimit:  1,
			expectedOffset: 0,
		},
		{
			name:           "mixed valid and invalid",
			queryParams:    "?limit=50&offset=abc",
			defaultLimit:   25,
			maxLimit:       100,
			expectedLimit:  50,
			expectedOffset: 0,
		},
		{
			name:           "mixed invalid and valid",
			queryParams:    "?limit=abc&offset=100",
			defaultLimit:   25,
			maxLimit:       100,
			expectedLimit:  25,
			expectedOffset: 100,
		},
		{
			name:           "float values",
			queryParams:    "?limit=25.5&offset=10.5",
			defaultLimit:   25,
			maxLimit:       100,
			expectedLimit:  25,
			expectedOffset: 0,
		},
		{
			name:           "negative string values",
			queryParams:    "?limit=-50&offset=-100",
			defaultLimit:   25,
			maxLimit:       100,
			expectedLimit:  25,
			expectedOffset: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/test"+tt.queryParams, nil)
			limit, offset := parsePagination(req, tt.defaultLimit, tt.maxLimit)

			assert.Equal(t, tt.expectedLimit, limit, "limit mismatch")
			assert.Equal(t, tt.expectedOffset, offset, "offset mismatch")
		})
	}
}

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
		},
		{
			name:      "zero limit",
			limit:     0,
			offset:    0,
			maxLimit:  100,
			expectErr: true,
		},
		{
			name:      "negative offset",
			limit:     25,
			offset:    -1,
			maxLimit:  100,
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validatePagination(tt.limit, tt.offset, tt.maxLimit)
			if tt.expectErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

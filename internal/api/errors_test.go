package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWriteJSON(t *testing.T) {
	tests := []struct {
		name           string
		statusCode     int
		data           interface{}
		expectedStatus int
		expectedBody   string
	}{
		{
			name:           "success response",
			statusCode:     http.StatusOK,
			data:           map[string]string{"message": "success"},
			expectedStatus: http.StatusOK,
			expectedBody:   `{"message":"success"}`,
		},
		{
			name:           "error response",
			statusCode:     http.StatusBadRequest,
			data:           ErrorResponse{Error: "bad request", Details: "invalid input"},
			expectedStatus: http.StatusBadRequest,
			expectedBody:   `{"error":"bad request","details":"invalid input"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			writeJSON(w, tt.statusCode, tt.data)

			assert.Equal(t, tt.expectedStatus, w.Code)
			assert.Equal(t, "application/json", w.Header().Get("Content-Type"))

			// Parse and compare JSON (ignoring whitespace differences)
			var expected, actual interface{}
			err := json.Unmarshal([]byte(tt.expectedBody), &expected)
			require.NoError(t, err)
			err = json.Unmarshal(w.Body.Bytes(), &actual)
			require.NoError(t, err)
			assert.Equal(t, expected, actual)
		})
	}
}

func TestWriteBadRequest(t *testing.T) {
	w := httptest.NewRecorder()
	writeBadRequest(w, "invalid parameter")

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Equal(t, "application/json", w.Header().Get("Content-Type"))

	var response ErrorResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.Equal(t, "Bad Request", response.Error)
	assert.Equal(t, "invalid parameter", response.Details)
}

func TestWriteNotFound(t *testing.T) {
	w := httptest.NewRecorder()
	writeNotFound(w, "resource not found")

	assert.Equal(t, http.StatusNotFound, w.Code)
	assert.Equal(t, "application/json", w.Header().Get("Content-Type"))

	var response ErrorResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.Equal(t, "Not Found", response.Error)
	assert.Equal(t, "resource not found", response.Details)
}

func TestWriteInternalError(t *testing.T) {
	w := httptest.NewRecorder()
	writeInternalError(w, errors.New("database connection failed"))

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.Equal(t, "application/json", w.Header().Get("Content-Type"))

	var response ErrorResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.Equal(t, "Internal Server Error", response.Error)
	// Should return generic message, not actual error details
	assert.Equal(t, "An unexpected error occurred", response.Details)
}

func TestWriteServiceUnavailable(t *testing.T) {
	w := httptest.NewRecorder()
	writeServiceUnavailable(w, "database offline")

	assert.Equal(t, http.StatusServiceUnavailable, w.Code)
	assert.Equal(t, "application/json", w.Header().Get("Content-Type"))

	var response ErrorResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.Equal(t, "Service Unavailable", response.Error)
	assert.Equal(t, "database offline", response.Details)
}

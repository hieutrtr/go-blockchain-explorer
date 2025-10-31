package api

import (
	"encoding/json"
	"net/http"

	"github.com/hieutt50/go-blockchain-explorer/internal/util"
)

// ErrorResponse represents a JSON error response
type ErrorResponse struct {
	Error   string `json:"error"`
	Details string `json:"details,omitempty"`
}

// writeJSON writes a JSON response with the given status code
func writeJSON(w http.ResponseWriter, statusCode int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	if err := json.NewEncoder(w).Encode(data); err != nil {
		util.Error("failed to encode JSON response", "error", err.Error())
	}
}

// writeError writes a JSON error response and logs the error
func writeError(w http.ResponseWriter, statusCode int, message string, details string) {
	// Log error with context
	util.Error("API error",
		"status", statusCode,
		"message", message,
		"details", details,
	)

	// Write error response
	writeJSON(w, statusCode, ErrorResponse{
		Error:   message,
		Details: details,
	})
}

// writeBadRequest writes a 400 Bad Request error
func writeBadRequest(w http.ResponseWriter, message string) {
	writeError(w, http.StatusBadRequest, "Bad Request", message)
}

// writeNotFound writes a 404 Not Found error
func writeNotFound(w http.ResponseWriter, message string) {
	writeError(w, http.StatusNotFound, "Not Found", message)
}

// writeInternalError writes a 500 Internal Server Error
func writeInternalError(w http.ResponseWriter, err error) {
	// Log the actual error with full context
	util.Error("internal server error", "error", err.Error())

	// Return generic message to client (no internal details)
	writeError(w, http.StatusInternalServerError, "Internal Server Error", "An unexpected error occurred")
}

// writeServiceUnavailable writes a 503 Service Unavailable error
func writeServiceUnavailable(w http.ResponseWriter, message string) {
	writeError(w, http.StatusServiceUnavailable, "Service Unavailable", message)
}

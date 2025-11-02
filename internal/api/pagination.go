package api

import (
	"net/http"
	"strconv"

	"github.com/hieutt50/go-blockchain-explorer/internal/util"
)

// parsePagination extracts and validates pagination parameters from the request
// Returns limit and offset with defaults and validation applied
func parsePagination(r *http.Request, defaultLimit, maxLimit int) (limit, offset int) {
	// Parse limit
	limitStr := r.URL.Query().Get("limit")
	if limitStr == "" {
		limit = defaultLimit
	} else {
		parsedLimit, err := strconv.Atoi(limitStr)
		if err != nil || parsedLimit < 1 {
			util.Warn("invalid pagination limit, using default",
				"provided", limitStr,
				"default", defaultLimit,
				"path", r.URL.Path)
			limit = defaultLimit
		} else if parsedLimit > maxLimit {
			util.Info("pagination limit exceeds maximum, clamping to max",
				"provided", parsedLimit,
				"max", maxLimit,
				"path", r.URL.Path)
			limit = maxLimit
		} else {
			limit = parsedLimit
		}
	}

	// Parse offset
	offsetStr := r.URL.Query().Get("offset")
	if offsetStr == "" {
		offset = 0
	} else {
		parsedOffset, err := strconv.Atoi(offsetStr)
		if err != nil || parsedOffset < 0 {
			util.Warn("invalid pagination offset, using zero",
				"provided", offsetStr,
				"path", r.URL.Path)
			offset = 0
		} else {
			offset = parsedOffset
		}
	}

	return limit, offset
}

// validatePagination checks if pagination parameters are valid
func validatePagination(limit, offset, maxLimit int) error {
	if limit < 1 {
		return ErrInvalidLimit
	}
	if limit > maxLimit {
		return ErrLimitTooLarge
	}
	if offset < 0 {
		return ErrInvalidOffset
	}
	return nil
}

// Pagination errors
var (
	ErrInvalidLimit   = newValidationError("limit must be greater than 0")
	ErrLimitTooLarge  = newValidationError("limit exceeds maximum allowed value")
	ErrInvalidOffset  = newValidationError("offset must be greater than or equal to 0")
)

// ValidationError represents a validation error
type ValidationError struct {
	message string
}

func newValidationError(message string) *ValidationError {
	return &ValidationError{message: message}
}

func (e *ValidationError) Error() string {
	return e.message
}

// NewPaginatedResponse creates a standardized paginated response with metadata
// data can be any type ([]Block, []Transaction, etc.)
// Returns a map with data, total, limit, and offset fields
func NewPaginatedResponse(data interface{}, total, limit, offset int) map[string]interface{} {
	return map[string]interface{}{
		"data":   data,
		"total":  total,
		"limit":  limit,
		"offset": offset,
	}
}

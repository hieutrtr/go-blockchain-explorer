package api

import (
	"net/http"
	"strconv"
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
			limit = defaultLimit
		} else if parsedLimit > maxLimit {
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

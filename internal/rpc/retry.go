package rpc

import (
	"context"
	"log/slog"
	"time"
)

// retryConfig holds retry-specific configuration
type retryConfig struct {
	maxRetries int
	baseDelay  time.Duration
}

// calculateBackoff calculates the exponential backoff delay for a given attempt
// Returns delays: 1s, 2s, 4s, 8s, 16s for attempts 0-4
func calculateBackoff(attempt int, baseDelay time.Duration) time.Duration {
	if attempt < 0 {
		return baseDelay
	}
	// Exponential backoff: baseDelay * 2^attempt
	multiplier := 1 << uint(attempt) // 2^attempt
	return baseDelay * time.Duration(multiplier)
}

// retryWithBackoff executes a function with exponential backoff retry logic
// It will retry up to maxRetries times for transient and rate limit errors
// Permanent errors fail immediately without retry
func retryWithBackoff(
	ctx context.Context,
	cfg *retryConfig,
	operation func() error,
	logger *slog.Logger,
	operationName string,
) error {
	var lastErr error

	for attempt := 0; attempt <= cfg.maxRetries; attempt++ {
		// Execute the operation
		err := operation()

		if err == nil {
			// Success
			if attempt > 0 {
				logger.Info("operation succeeded after retry",
					"operation", operationName,
					"attempt", attempt+1,
					"total_attempts", attempt+1,
				)
			}
			return nil
		}

		lastErr = err

		// Classify the error
		errorType := classifyError(err)

		// Log the error
		logger.Warn("operation failed",
			"operation", operationName,
			"attempt", attempt+1,
			"error_type", errorType.String(),
			"error", err.Error(),
		)

		// If permanent error, fail immediately without retry
		if errorType == ErrPermanent {
			logger.Error("permanent error detected, not retrying",
				"operation", operationName,
				"error", err.Error(),
			)
			return NewRPCError("permanent error, not retrying", err)
		}

		// If we've exhausted retries, return the error
		if attempt >= cfg.maxRetries {
			logger.Error("max retries exceeded",
				"operation", operationName,
				"max_retries", cfg.maxRetries,
				"error", err.Error(),
			)
			return NewRPCError("max retries exceeded", err)
		}

		// Calculate backoff delay
		backoffDelay := calculateBackoff(attempt, cfg.baseDelay)

		logger.Info("retrying after backoff",
			"operation", operationName,
			"attempt", attempt+1,
			"next_attempt", attempt+2,
			"backoff_duration", backoffDelay.String(),
			"error_type", errorType.String(),
		)

		// Wait for backoff duration or context cancellation
		select {
		case <-time.After(backoffDelay):
			// Continue to next retry
		case <-ctx.Done():
			// Context cancelled, return immediately
			logger.Info("retry cancelled by context",
				"operation", operationName,
				"attempt", attempt+1,
			)
			return ctx.Err()
		}
	}

	// Should not reach here, but return last error just in case
	return lastErr
}

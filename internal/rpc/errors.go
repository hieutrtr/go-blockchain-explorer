package rpc

import (
	"errors"
	"fmt"
	"net"
	"strings"
	"syscall"
)

// ErrorType represents the category of an RPC error
type ErrorType int

const (
	// ErrTransient represents temporary errors that should be retried (network issues, timeouts)
	ErrTransient ErrorType = iota

	// ErrPermanent represents errors that should not be retried (invalid parameters, method not found)
	ErrPermanent

	// ErrRateLimit represents rate limiting errors (HTTP 429, quota exceeded)
	ErrRateLimit
)

// String returns the string representation of ErrorType
func (e ErrorType) String() string {
	switch e {
	case ErrTransient:
		return "transient"
	case ErrPermanent:
		return "permanent"
	case ErrRateLimit:
		return "rate_limit"
	default:
		return "unknown"
	}
}

// classifyError analyzes an error and determines if it should be retried
func classifyError(err error) ErrorType {
	if err == nil {
		return ErrPermanent // shouldn't happen, but treat as permanent
	}

	errStr := err.Error()
	errStrLower := strings.ToLower(errStr)

	// Check for rate limiting errors
	if strings.Contains(errStrLower, "429") ||
		strings.Contains(errStrLower, "too many requests") ||
		strings.Contains(errStrLower, "rate limit") ||
		strings.Contains(errStrLower, "quota exceeded") {
		return ErrRateLimit
	}

	// Check for network errors (transient)
	var netErr net.Error
	if errors.As(err, &netErr) {
		// Network timeouts and temporary errors are transient
		if netErr.Timeout() || netErr.Temporary() {
			return ErrTransient
		}
	}

	// Check for connection refused (transient - server might be temporarily down)
	var syscallErr *net.OpError
	if errors.As(err, &syscallErr) {
		if errors.Is(syscallErr.Err, syscall.ECONNREFUSED) {
			return ErrTransient
		}
	}

	// Check for context timeouts (transient)
	if strings.Contains(errStrLower, "context deadline exceeded") ||
		strings.Contains(errStrLower, "context canceled") {
		return ErrTransient
	}

	// Check for DNS errors (transient - might be temporary DNS issues)
	if strings.Contains(errStrLower, "no such host") ||
		strings.Contains(errStrLower, "dns") {
		return ErrTransient
	}

	// Check for connection errors (transient)
	if strings.Contains(errStrLower, "connection reset") ||
		strings.Contains(errStrLower, "broken pipe") ||
		strings.Contains(errStrLower, "connection refused") {
		return ErrTransient
	}

	// Check for permanent errors (invalid parameters, method not found)
	if strings.Contains(errStrLower, "invalid") ||
		strings.Contains(errStrLower, "method not found") ||
		strings.Contains(errStrLower, "missing required") ||
		strings.Contains(errStrLower, "malformed") ||
		strings.Contains(errStrLower, "parse error") {
		return ErrPermanent
	}

	// Check for EOF and unexpected EOF (can be transient network issues)
	if strings.Contains(errStrLower, "eof") {
		return ErrTransient
	}

	// Default to transient for unknown errors (safer to retry)
	// Better to retry unnecessarily than fail on a recoverable error
	return ErrTransient
}

// RPCError wraps an error with additional context
type RPCError struct {
	Type    ErrorType
	Message string
	Err     error
}

// Error implements the error interface
func (e *RPCError) Error() string {
	return fmt.Sprintf("[%s] %s: %v", e.Type, e.Message, e.Err)
}

// Unwrap returns the underlying error
func (e *RPCError) Unwrap() error {
	return e.Err
}

// NewRPCError creates a new RPCError with error classification
func NewRPCError(message string, err error) *RPCError {
	return &RPCError{
		Type:    classifyError(err),
		Message: message,
		Err:     err,
	}
}

// errorTypeToMetricsLabel converts internal ErrorType to metrics label value
// Maps to: network, rate_limit, invalid_param, timeout, other
func errorTypeToMetricsLabel(errType ErrorType) string {
	switch errType {
	case ErrRateLimit:
		return "rate_limit"
	case ErrPermanent:
		return "invalid_param"
	case ErrTransient:
		// For transient errors, we need to distinguish between network and timeout
		// For now, we map all transient errors to "network" since the main causes are network-related
		// Future improvement: pass the error itself to classify as timeout vs network
		return "network"
	default:
		return "other"
	}
}

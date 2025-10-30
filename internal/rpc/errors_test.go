package rpc

import (
	"context"
	"errors"
	"net"
	"syscall"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestErrorType_String(t *testing.T) {
	tests := []struct {
		name     string
		errType  ErrorType
		expected string
	}{
		{"transient", ErrTransient, "transient"},
		{"permanent", ErrPermanent, "permanent"},
		{"rate_limit", ErrRateLimit, "rate_limit"},
		{"unknown", ErrorType(999), "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.errType.String()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestClassifyError_RateLimit(t *testing.T) {
	tests := []struct {
		name string
		err  error
	}{
		{"429 error", errors.New("HTTP 429: too many requests")},
		{"rate limit text", errors.New("rate limit exceeded")},
		{"quota exceeded", errors.New("quota exceeded for this API key")},
		{"mixed case", errors.New("Rate Limit Reached")},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := classifyError(tt.err)
			assert.Equal(t, ErrRateLimit, result, "should be classified as rate limit")
		})
	}
}

func TestClassifyError_Permanent(t *testing.T) {
	tests := []struct {
		name string
		err  error
	}{
		{"invalid parameter", errors.New("invalid block height parameter")},
		{"method not found", errors.New("method eth_invalidMethod not found")},
		{"missing required", errors.New("missing required field: address")},
		{"malformed", errors.New("malformed JSON-RPC request")},
		{"parse error", errors.New("parse error: invalid hex string")},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := classifyError(tt.err)
			assert.Equal(t, ErrPermanent, result, "should be classified as permanent")
		})
	}
}

func TestClassifyError_Transient(t *testing.T) {
	tests := []struct {
		name string
		err  error
	}{
		{"connection reset", errors.New("connection reset by peer")},
		{"connection refused", errors.New("connection refused")},
		{"broken pipe", errors.New("write: broken pipe")},
		{"no such host", errors.New("no such host: invalid.domain")},
		{"dns error", errors.New("DNS lookup failed")},
		{"context deadline", errors.New("context deadline exceeded")},
		{"context canceled", errors.New("context canceled")},
		{"eof", errors.New("unexpected EOF")},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := classifyError(tt.err)
			assert.Equal(t, ErrTransient, result, "should be classified as transient")
		})
	}
}

func TestClassifyError_NetworkErrors(t *testing.T) {
	t.Run("timeout error", func(t *testing.T) {
		// Create a network timeout error
		timeoutErr := &testNetError{timeout: true, temporary: false}
		result := classifyError(timeoutErr)
		assert.Equal(t, ErrTransient, result, "timeout should be transient")
	})

	t.Run("temporary error", func(t *testing.T) {
		// Create a temporary network error
		tempErr := &testNetError{timeout: false, temporary: true}
		result := classifyError(tempErr)
		assert.Equal(t, ErrTransient, result, "temporary error should be transient")
	})

	t.Run("connection refused", func(t *testing.T) {
		// Create a connection refused error
		opErr := &net.OpError{
			Op:  "dial",
			Net: "tcp",
			Err: syscall.ECONNREFUSED,
		}
		result := classifyError(opErr)
		assert.Equal(t, ErrTransient, result, "connection refused should be transient")
	})
}

func TestClassifyError_NilError(t *testing.T) {
	result := classifyError(nil)
	// Nil error treated as permanent (shouldn't happen in practice)
	assert.Equal(t, ErrPermanent, result)
}

func TestClassifyError_UnknownError(t *testing.T) {
	// Unknown errors default to transient (safer to retry)
	unknownErr := errors.New("some unknown error message")
	result := classifyError(unknownErr)
	assert.Equal(t, ErrTransient, result, "unknown errors should default to transient")
}

func TestRPCError(t *testing.T) {
	t.Run("basic rpc error", func(t *testing.T) {
		underlyingErr := errors.New("connection timeout")
		rpcErr := NewRPCError("failed to fetch block", underlyingErr)

		assert.Equal(t, ErrTransient, rpcErr.Type)
		assert.Equal(t, "failed to fetch block", rpcErr.Message)
		assert.Equal(t, underlyingErr, rpcErr.Err)
	})

	t.Run("error method", func(t *testing.T) {
		underlyingErr := errors.New("invalid parameter")
		rpcErr := NewRPCError("bad request", underlyingErr)

		errorStr := rpcErr.Error()
		assert.Contains(t, errorStr, "permanent")
		assert.Contains(t, errorStr, "bad request")
		assert.Contains(t, errorStr, "invalid parameter")
	})

	t.Run("unwrap", func(t *testing.T) {
		underlyingErr := errors.New("test error")
		rpcErr := NewRPCError("wrapper", underlyingErr)

		unwrapped := rpcErr.Unwrap()
		assert.Equal(t, underlyingErr, unwrapped)
	})
}

// testNetError is a helper type that implements net.Error for testing
type testNetError struct {
	timeout   bool
	temporary bool
}

func (e *testNetError) Error() string   { return "test network error" }
func (e *testNetError) Timeout() bool   { return e.timeout }
func (e *testNetError) Temporary() bool { return e.temporary }

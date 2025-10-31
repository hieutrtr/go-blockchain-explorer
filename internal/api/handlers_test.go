package api

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestValidateAddress(t *testing.T) {
	tests := []struct {
		name    string
		address string
		valid   bool
	}{
		{
			name:    "valid address",
			address: "0x742d35Cc6634C0532925a3b844Bc9e7595f0bEb0",
			valid:   true,
		},
		{
			name:    "valid lowercase address",
			address: "0x742d35cc6634c0532925a3b844bc9e7595f0beb0",
			valid:   true,
		},
		{
			name:    "missing 0x prefix",
			address: "742d35Cc6634C0532925a3b844Bc9e7595f0bEb0",
			valid:   false,
		},
		{
			name:    "too short",
			address: "0x742d35Cc6634C0532925a3b844Bc9e7595f0b",
			valid:   false,
		},
		{
			name:    "too long",
			address: "0x742d35Cc6634C0532925a3b844Bc9e7595f0bEb0FF",
			valid:   false,
		},
		{
			name:    "invalid characters",
			address: "0x742d35Cc6634C0532925a3b844Bc9e7595f0bEbG",
			valid:   false,
		},
		{
			name:    "empty string",
			address: "",
			valid:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := validateAddress(tt.address)
			assert.Equal(t, tt.valid, result)
		})
	}
}

func TestValidateHash(t *testing.T) {
	tests := []struct {
		name  string
		hash  string
		valid bool
	}{
		{
			name:  "valid tx hash",
			hash:  "0x1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef",
			valid: true,
		},
		{
			name:  "valid block hash",
			hash:  "0xABCDEF1234567890ABCDEF1234567890ABCDEF1234567890ABCDEF1234567890",
			valid: true,
		},
		{
			name:  "missing 0x prefix",
			hash:  "1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef",
			valid: false,
		},
		{
			name:  "too short",
			hash:  "0x1234567890abcdef",
			valid: false,
		},
		{
			name:  "too long",
			hash:  "0x1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdefff",
			valid: false,
		},
		{
			name:  "invalid characters",
			hash:  "0x1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdeg",
			valid: false,
		},
		{
			name:  "empty string",
			hash:  "",
			valid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := validateHash(tt.hash)
			assert.Equal(t, tt.valid, result)
		})
	}
}

func TestParseHexBytes(t *testing.T) {
	tests := []struct {
		name      string
		hexStr    string
		expectErr bool
		expected  []byte
	}{
		{
			name:      "valid hex string",
			hexStr:    "0x1234",
			expectErr: false,
			expected:  []byte{0x12, 0x34},
		},
		{
			name:      "valid empty hex",
			hexStr:    "0x",
			expectErr: false,
			expected:  []byte{},
		},
		{
			name:      "missing 0x prefix",
			hexStr:    "1234",
			expectErr: true,
		},
		{
			name:      "invalid hex characters",
			hexStr:    "0x12GH",
			expectErr: true,
		},
		{
			name:      "odd length hex",
			hexStr:    "0x123",
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := parseHexBytes(tt.hexStr)
			if tt.expectErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}

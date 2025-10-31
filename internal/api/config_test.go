package api

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewConfig(t *testing.T) {
	tests := []struct {
		name             string
		envVars          map[string]string
		expectedPort     int
		expectedOrigins  string
	}{
		{
			name:            "default values",
			envVars:         map[string]string{},
			expectedPort:    8080,
			expectedOrigins: "*",
		},
		{
			name: "custom port",
			envVars: map[string]string{
				"API_PORT": "3000",
			},
			expectedPort:    3000,
			expectedOrigins: "*",
		},
		{
			name: "custom CORS origins",
			envVars: map[string]string{
				"API_CORS_ORIGINS": "https://example.com",
			},
			expectedPort:    8080,
			expectedOrigins: "https://example.com",
		},
		{
			name: "invalid port - use default",
			envVars: map[string]string{
				"API_PORT": "invalid",
			},
			expectedPort:    8080,
			expectedOrigins: "*",
		},
		{
			name: "port out of range - use default",
			envVars: map[string]string{
				"API_PORT": "99999",
			},
			expectedPort:    8080,
			expectedOrigins: "*",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clear environment
			os.Clearenv()

			// Set test environment variables
			for k, v := range tt.envVars {
				os.Setenv(k, v)
			}

			config := NewConfig()

			assert.Equal(t, tt.expectedPort, config.Port)
			assert.Equal(t, tt.expectedOrigins, config.CORSOrigins)
		})
	}

	// Clean up
	os.Clearenv()
}

func TestConfigAddress(t *testing.T) {
	config := &Config{Port: 3000}
	assert.Equal(t, ":3000", config.Address())

	config.Port = 8080
	assert.Equal(t, ":8080", config.Address())
}

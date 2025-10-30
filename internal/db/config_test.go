package db

import (
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewConfig_Success(t *testing.T) {
	// Set required environment variables
	os.Setenv("DB_NAME", "test_db")
	os.Setenv("DB_USER", "test_user")
	os.Setenv("DB_PASSWORD", "test_password")
	defer func() {
		os.Unsetenv("DB_NAME")
		os.Unsetenv("DB_USER")
		os.Unsetenv("DB_PASSWORD")
	}()

	config, err := NewConfig()
	require.NoError(t, err)
	require.NotNil(t, config)

	assert.Equal(t, "localhost", config.Host)
	assert.Equal(t, 5432, config.Port)
	assert.Equal(t, "test_db", config.Name)
	assert.Equal(t, "test_user", config.User)
	assert.Equal(t, "test_password", config.Password)
	assert.Equal(t, 20, config.MaxConns)
}

func TestNewConfig_WithOptionalEnvVars(t *testing.T) {
	// Set all environment variables
	os.Setenv("DB_HOST", "db.example.com")
	os.Setenv("DB_PORT", "5433")
	os.Setenv("DB_NAME", "production_db")
	os.Setenv("DB_USER", "prod_user")
	os.Setenv("DB_PASSWORD", "prod_password")
	os.Setenv("DB_MAX_CONNS", "50")
	defer func() {
		os.Unsetenv("DB_HOST")
		os.Unsetenv("DB_PORT")
		os.Unsetenv("DB_NAME")
		os.Unsetenv("DB_USER")
		os.Unsetenv("DB_PASSWORD")
		os.Unsetenv("DB_MAX_CONNS")
	}()

	config, err := NewConfig()
	require.NoError(t, err)
	require.NotNil(t, config)

	assert.Equal(t, "db.example.com", config.Host)
	assert.Equal(t, 5433, config.Port)
	assert.Equal(t, "production_db", config.Name)
	assert.Equal(t, "prod_user", config.User)
	assert.Equal(t, "prod_password", config.Password)
	assert.Equal(t, 50, config.MaxConns)
}

func TestNewConfig_MissingDBName(t *testing.T) {
	os.Unsetenv("DB_NAME")
	os.Setenv("DB_USER", "test_user")
	os.Setenv("DB_PASSWORD", "test_password")
	defer func() {
		os.Unsetenv("DB_USER")
		os.Unsetenv("DB_PASSWORD")
	}()

	config, err := NewConfig()
	assert.Error(t, err)
	assert.Nil(t, config)
	assert.Contains(t, err.Error(), "DB_NAME")
}

func TestNewConfig_MissingDBUser(t *testing.T) {
	os.Setenv("DB_NAME", "test_db")
	os.Unsetenv("DB_USER")
	os.Setenv("DB_PASSWORD", "test_password")
	defer func() {
		os.Unsetenv("DB_NAME")
		os.Unsetenv("DB_PASSWORD")
	}()

	config, err := NewConfig()
	assert.Error(t, err)
	assert.Nil(t, config)
	assert.Contains(t, err.Error(), "DB_USER")
}

func TestNewConfig_MissingDBPassword(t *testing.T) {
	os.Setenv("DB_NAME", "test_db")
	os.Setenv("DB_USER", "test_user")
	os.Unsetenv("DB_PASSWORD")
	defer func() {
		os.Unsetenv("DB_NAME")
		os.Unsetenv("DB_USER")
	}()

	config, err := NewConfig()
	assert.Error(t, err)
	assert.Nil(t, config)
	assert.Contains(t, err.Error(), "DB_PASSWORD")
}

func TestNewConfig_InvalidPort(t *testing.T) {
	os.Setenv("DB_NAME", "test_db")
	os.Setenv("DB_USER", "test_user")
	os.Setenv("DB_PASSWORD", "test_password")
	os.Setenv("DB_PORT", "invalid")
	defer func() {
		os.Unsetenv("DB_NAME")
		os.Unsetenv("DB_USER")
		os.Unsetenv("DB_PASSWORD")
		os.Unsetenv("DB_PORT")
	}()

	config, err := NewConfig()
	assert.Error(t, err)
	assert.Nil(t, config)
	assert.Contains(t, err.Error(), "DB_PORT")
}

func TestNewConfig_PortOutOfRange(t *testing.T) {
	tests := []struct {
		name string
		port string
	}{
		{"port too low", "0"},
		{"port too high", "65536"},
		{"negative port", "-1"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			os.Setenv("DB_NAME", "test_db")
			os.Setenv("DB_USER", "test_user")
			os.Setenv("DB_PASSWORD", "test_password")
			os.Setenv("DB_PORT", tt.port)
			defer func() {
				os.Unsetenv("DB_NAME")
				os.Unsetenv("DB_USER")
				os.Unsetenv("DB_PASSWORD")
				os.Unsetenv("DB_PORT")
			}()

			config, err := NewConfig()
			assert.Error(t, err)
			assert.Nil(t, config)
		})
	}
}

func TestNewConfig_InvalidMaxConns(t *testing.T) {
	tests := []struct {
		name     string
		maxConns string
	}{
		{"invalid format", "invalid"},
		{"zero connections", "0"},
		{"negative connections", "-1"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			os.Setenv("DB_NAME", "test_db")
			os.Setenv("DB_USER", "test_user")
			os.Setenv("DB_PASSWORD", "test_password")
			os.Setenv("DB_MAX_CONNS", tt.maxConns)
			defer func() {
				os.Unsetenv("DB_NAME")
				os.Unsetenv("DB_USER")
				os.Unsetenv("DB_PASSWORD")
				os.Unsetenv("DB_MAX_CONNS")
			}()

			config, err := NewConfig()
			assert.Error(t, err)
			assert.Nil(t, config)
		})
	}
}

func TestNewConfigWithDefaults(t *testing.T) {
	config := NewConfigWithDefaults("testhost", 5433, "testdb", "testuser", "testpass", 10)
	require.NotNil(t, config)

	assert.Equal(t, "testhost", config.Host)
	assert.Equal(t, 5433, config.Port)
	assert.Equal(t, "testdb", config.Name)
	assert.Equal(t, "testuser", config.User)
	assert.Equal(t, "testpass", config.Password)
	assert.Equal(t, 10, config.MaxConns)
}

func TestConfig_ConnectionString(t *testing.T) {
	config := NewConfigWithDefaults("localhost", 5432, "mydb", "myuser", "mypass", 20)
	connStr := config.ConnectionString()

	assert.Equal(t, "postgres://myuser:mypass@localhost:5432/mydb?sslmode=disable", connStr)
}

func TestConfig_SafeString(t *testing.T) {
	config := NewConfigWithDefaults("localhost", 5432, "mydb", "myuser", "secret_password", 20)
	safeStr := config.SafeString()

	assert.Contains(t, safeStr, "myuser")
	assert.Contains(t, safeStr, "localhost")
	assert.Contains(t, safeStr, "5432")
	assert.Contains(t, safeStr, "mydb")
	assert.Contains(t, safeStr, "maxConns=20")
	assert.NotContains(t, safeStr, "secret_password")
	assert.Contains(t, safeStr, "***")
}

func TestNewConfigWithDefaults_AllFields(t *testing.T) {
	config := NewConfigWithDefaults("testhost", 5433, "testdb", "testuser", "testpass", 10)

	assert.Equal(t, "testhost", config.Host)
	assert.Equal(t, 5433, config.Port)
	assert.Equal(t, "testdb", config.Name)
	assert.Equal(t, "testuser", config.User)
	assert.Equal(t, "testpass", config.Password)
	assert.Equal(t, 10, config.MaxConns)
	assert.Equal(t, 5*time.Second, config.ConnTimeout)
	assert.Equal(t, 5*time.Minute, config.IdleTimeout)
	assert.Equal(t, 30*time.Minute, config.ConnLifetime)
}

func TestNewConfigWithDefaults_TimeoutDefaults(t *testing.T) {
	config := NewConfigWithDefaults("host", 5432, "db", "user", "pass", 20)

	// Verify default timeout values are set correctly
	assert.Equal(t, 5*time.Second, config.ConnTimeout)
	assert.Equal(t, 5*time.Minute, config.IdleTimeout)
	assert.Equal(t, 30*time.Minute, config.ConnLifetime)
}

func TestConfig_ConnectionString_SpecialCharacters(t *testing.T) {
	config := NewConfigWithDefaults("localhost", 5432, "my-db", "my_user", "p@ss:word!", 20)
	connStr := config.ConnectionString()

	// Connection string should include special characters (pgx will handle encoding)
	assert.Contains(t, connStr, "my_user")
	assert.Contains(t, connStr, "p@ss:word!")
	assert.Contains(t, connStr, "my-db")
}

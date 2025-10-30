#!/bin/bash
set -e

# Export environment variables for integration tests
export DB_HOST=localhost
export DB_PORT=5432
export DB_NAME=test_migrations
export DB_USER=postgres
export DB_PASSWORD=postgres

echo "Running integration tests with PostgreSQL..."
go test -v -count=1 ./internal/db/... -run Integration

echo ""
echo "Running full test suite with coverage..."
go test -count=1 -coverprofile=coverage.out ./internal/db/...
go tool cover -func=coverage.out | grep total

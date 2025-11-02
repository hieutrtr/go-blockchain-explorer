#!/bin/bash

# Blockchain Explorer - Startup Script
# This script starts both the API server and worker processes

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Get the project root directory
PROJECT_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$PROJECT_ROOT"

echo -e "${GREEN}=== Blockchain Explorer Startup ===${NC}"
echo ""

# Check if .env file exists
if [ ! -f .env ]; then
    echo -e "${RED}Error: .env file not found${NC}"
    echo "Please copy .env.example to .env and configure your settings:"
    echo "  cp .env.example .env"
    exit 1
fi

# Load environment variables from .env file
echo -e "${YELLOW}Loading environment variables from .env...${NC}"
export $(grep -v '^#' .env | xargs)

# Validate required environment variables
echo -e "${YELLOW}Validating configuration...${NC}"

REQUIRED_VARS=("DB_NAME" "DB_USER" "DB_PASSWORD" "RPC_URL")
MISSING_VARS=()

for var in "${REQUIRED_VARS[@]}"; do
    if [ -z "${!var}" ]; then
        MISSING_VARS+=("$var")
    fi
done

if [ ${#MISSING_VARS[@]} -ne 0 ]; then
    echo -e "${RED}Error: Missing required environment variables:${NC}"
    for var in "${MISSING_VARS[@]}"; do
        echo "  - $var"
    done
    echo ""
    echo "Please configure these variables in your .env file"
    exit 1
fi

# Check if PostgreSQL is accessible
echo -e "${YELLOW}Checking database connection...${NC}"
DB_HOST=${DB_HOST:-localhost}
DB_PORT=${DB_PORT:-5432}

if command -v psql &> /dev/null; then
    if ! PGPASSWORD=$DB_PASSWORD psql -h $DB_HOST -p $DB_PORT -U $DB_USER -d $DB_NAME -c '\q' 2>/dev/null; then
        echo -e "${RED}Warning: Cannot connect to database${NC}"
        echo "  Host: $DB_HOST:$DB_PORT"
        echo "  Database: $DB_NAME"
        echo "  User: $DB_USER"
        echo ""
        echo "Make sure PostgreSQL is running and credentials are correct"
        echo -e "${YELLOW}Continuing anyway...${NC}"
        echo ""
    else
        echo -e "${GREEN}✓ Database connection successful${NC}"
    fi
else
    echo -e "${YELLOW}Warning: psql not found, skipping database check${NC}"
fi

# Create log directory
mkdir -p logs

# PID file locations
API_PID_FILE="logs/api.pid"
WORKER_PID_FILE="logs/worker.pid"

# Check if processes are already running
if [ -f "$API_PID_FILE" ] && kill -0 $(cat "$API_PID_FILE") 2>/dev/null; then
    echo -e "${YELLOW}API server is already running (PID: $(cat $API_PID_FILE))${NC}"
    read -p "Stop and restart? (y/N) " -n 1 -r
    echo
    if [[ $REPLY =~ ^[Yy]$ ]]; then
        kill $(cat "$API_PID_FILE")
        rm -f "$API_PID_FILE"
        sleep 2
    else
        echo "Skipping API server startup"
        API_SKIP=1
    fi
fi

if [ -f "$WORKER_PID_FILE" ] && kill -0 $(cat "$WORKER_PID_FILE") 2>/dev/null; then
    echo -e "${YELLOW}Worker is already running (PID: $(cat $WORKER_PID_FILE))${NC}"
    read -p "Stop and restart? (y/N) " -n 1 -r
    echo
    if [[ $REPLY =~ ^[Yy]$ ]]; then
        kill $(cat "$WORKER_PID_FILE")
        rm -f "$WORKER_PID_FILE"
        sleep 2
    else
        echo "Skipping worker startup"
        WORKER_SKIP=1
    fi
fi

echo ""

# Start API Server
if [ -z "$API_SKIP" ]; then
    echo -e "${YELLOW}Starting API server...${NC}"
    nohup go run cmd/api/main.go > logs/api.log 2>&1 &
    API_PID=$!
    echo $API_PID > "$API_PID_FILE"

    # Wait a moment and check if it's still running
    sleep 2
    if kill -0 $API_PID 2>/dev/null; then
        API_PORT=${API_PORT:-8080}
        echo -e "${GREEN}✓ API server started${NC} (PID: $API_PID, Port: $API_PORT)"
        echo "  Logs: logs/api.log"
        echo "  Health: http://localhost:$API_PORT/health"
    else
        echo -e "${RED}✗ API server failed to start${NC}"
        echo "Check logs/api.log for errors"
        rm -f "$API_PID_FILE"
    fi
fi

# Start Worker
if [ -z "$WORKER_SKIP" ]; then
    echo -e "${YELLOW}Starting worker...${NC}"
    nohup go run cmd/worker/main.go > logs/worker.log 2>&1 &
    WORKER_PID=$!
    echo $WORKER_PID > "$WORKER_PID_FILE"

    # Wait a moment and check if it's still running
    sleep 2
    if kill -0 $WORKER_PID 2>/dev/null; then
        echo -e "${GREEN}✓ Worker started${NC} (PID: $WORKER_PID)"
        echo "  Logs: logs/worker.log"
        echo "  Metrics: http://localhost:9090/metrics"
    else
        echo -e "${RED}✗ Worker failed to start${NC}"
        echo "Check logs/worker.log for errors"
        rm -f "$WORKER_PID_FILE"
    fi
fi

echo ""
echo -e "${GREEN}=== Startup Complete ===${NC}"
echo ""
echo "Running services:"
if [ -f "$API_PID_FILE" ] && kill -0 $(cat "$API_PID_FILE") 2>/dev/null; then
    echo "  • API Server (PID: $(cat $API_PID_FILE))"
fi
if [ -f "$WORKER_PID_FILE" ] && kill -0 $(cat "$WORKER_PID_FILE") 2>/dev/null; then
    echo "  • Worker (PID: $(cat $WORKER_PID_FILE))"
fi

echo ""
echo "Commands:"
echo "  • View API logs:    tail -f logs/api.log"
echo "  • View worker logs: tail -f logs/worker.log"
echo "  • Stop services:    ./scripts/stop.sh"
echo "  • Check status:     ./scripts/status.sh"

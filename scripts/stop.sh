#!/bin/bash

# Blockchain Explorer - Stop Script
# This script stops both the API server and worker processes

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Get the project root directory
PROJECT_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$PROJECT_ROOT"

echo -e "${YELLOW}=== Stopping Blockchain Explorer ===${NC}"
echo ""

API_PID_FILE="logs/api.pid"
WORKER_PID_FILE="logs/worker.pid"

STOPPED=0

# Stop API Server
if [ -f "$API_PID_FILE" ]; then
    API_PID=$(cat "$API_PID_FILE")
    if kill -0 $API_PID 2>/dev/null; then
        echo -e "${YELLOW}Stopping API server (PID: $API_PID)...${NC}"
        kill $API_PID

        # Wait for graceful shutdown (max 10 seconds)
        for i in {1..10}; do
            if ! kill -0 $API_PID 2>/dev/null; then
                break
            fi
            sleep 1
        done

        # Force kill if still running
        if kill -0 $API_PID 2>/dev/null; then
            echo -e "${YELLOW}Forcing stop...${NC}"
            kill -9 $API_PID 2>/dev/null || true
        fi

        echo -e "${GREEN}✓ API server stopped${NC}"
        STOPPED=1
    else
        echo -e "${YELLOW}API server is not running${NC}"
    fi
    rm -f "$API_PID_FILE"
else
    echo -e "${YELLOW}API server PID file not found${NC}"
fi

# Stop Worker
if [ -f "$WORKER_PID_FILE" ]; then
    WORKER_PID=$(cat "$WORKER_PID_FILE")
    if kill -0 $WORKER_PID 2>/dev/null; then
        echo -e "${YELLOW}Stopping worker (PID: $WORKER_PID)...${NC}"
        kill $WORKER_PID

        # Wait for graceful shutdown (max 10 seconds)
        for i in {1..10}; do
            if ! kill -0 $WORKER_PID 2>/dev/null; then
                break
            fi
            sleep 1
        done

        # Force kill if still running
        if kill -0 $WORKER_PID 2>/dev/null; then
            echo -e "${YELLOW}Forcing stop...${NC}"
            kill -9 $WORKER_PID 2>/dev/null || true
        fi

        echo -e "${GREEN}✓ Worker stopped${NC}"
        STOPPED=1
    else
        echo -e "${YELLOW}Worker is not running${NC}"
    fi
    rm -f "$WORKER_PID_FILE"
else
    echo -e "${YELLOW}Worker PID file not found${NC}"
fi

echo ""
if [ $STOPPED -eq 1 ]; then
    echo -e "${GREEN}All services stopped${NC}"
else
    echo -e "${YELLOW}No services were running${NC}"
fi

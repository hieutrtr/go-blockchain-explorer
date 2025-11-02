#!/bin/bash

# Blockchain Explorer - Status Script
# This script checks the status of API server and worker processes

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Get the project root directory
PROJECT_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$PROJECT_ROOT"

echo -e "${GREEN}=== Blockchain Explorer Status ===${NC}"
echo ""

API_PID_FILE="logs/api.pid"
WORKER_PID_FILE="logs/worker.pid"

# Check API Server
echo -e "${YELLOW}API Server:${NC}"
if [ -f "$API_PID_FILE" ]; then
    API_PID=$(cat "$API_PID_FILE")
    if kill -0 $API_PID 2>/dev/null; then
        echo -e "  Status: ${GREEN}Running${NC}"
        echo "  PID: $API_PID"

        # Get uptime
        if command -v ps &> /dev/null; then
            UPTIME=$(ps -o etime= -p $API_PID 2>/dev/null | tr -d ' ')
            echo "  Uptime: $UPTIME"
        fi

        # Check if API is responding
        API_PORT=${API_PORT:-8080}
        if command -v curl &> /dev/null; then
            if curl -s "http://localhost:$API_PORT/health" > /dev/null 2>&1; then
                echo -e "  Health: ${GREEN}OK${NC}"
                echo "  URL: http://localhost:$API_PORT"
            else
                echo -e "  Health: ${RED}Not responding${NC}"
            fi
        fi

        # Log file info
        if [ -f "logs/api.log" ]; then
            LOG_SIZE=$(du -h logs/api.log | cut -f1)
            RECENT_ERRORS=$(grep -i error logs/api.log 2>/dev/null | tail -3)
            echo "  Log: logs/api.log ($LOG_SIZE)"
            if [ ! -z "$RECENT_ERRORS" ]; then
                echo -e "  ${RED}Recent errors found in log${NC}"
            fi
        fi
    else
        echo -e "  Status: ${RED}Not running${NC} (stale PID file)"
    fi
else
    echo -e "  Status: ${RED}Not running${NC}"
fi

echo ""

# Check Worker
echo -e "${YELLOW}Worker:${NC}"
if [ -f "$WORKER_PID_FILE" ]; then
    WORKER_PID=$(cat "$WORKER_PID_FILE")
    if kill -0 $WORKER_PID 2>/dev/null; then
        echo -e "  Status: ${GREEN}Running${NC}"
        echo "  PID: $WORKER_PID"

        # Get uptime
        if command -v ps &> /dev/null; then
            UPTIME=$(ps -o etime= -p $WORKER_PID 2>/dev/null | tr -d ' ')
            echo "  Uptime: $UPTIME"
        fi

        # Check if metrics endpoint is responding
        if command -v curl &> /dev/null; then
            if curl -s "http://localhost:9090/metrics" > /dev/null 2>&1; then
                echo -e "  Metrics: ${GREEN}OK${NC}"
                echo "  URL: http://localhost:9090/metrics"
            else
                echo -e "  Metrics: ${YELLOW}Not responding${NC}"
            fi
        fi

        # Log file info
        if [ -f "logs/worker.log" ]; then
            LOG_SIZE=$(du -h logs/worker.log | cut -f1)
            RECENT_ERRORS=$(grep -i error logs/worker.log 2>/dev/null | tail -3)
            echo "  Log: logs/worker.log ($LOG_SIZE)"
            if [ ! -z "$RECENT_ERRORS" ]; then
                echo -e "  ${RED}Recent errors found in log${NC}"
            fi
        fi
    else
        echo -e "  Status: ${RED}Not running${NC} (stale PID file)"
    fi
else
    echo -e "  Status: ${RED}Not running${NC}"
fi

echo ""

# System resources
echo -e "${YELLOW}System Resources:${NC}"
if command -v df &> /dev/null; then
    DISK_USAGE=$(df -h . | awk 'NR==2 {print $5}')
    echo "  Disk Usage: $DISK_USAGE"
fi

if command -v free &> /dev/null; then
    MEM_USAGE=$(free -h | awk 'NR==2 {print $3 "/" $2}')
    echo "  Memory Usage: $MEM_USAGE"
fi

echo ""
echo "Commands:"
echo "  • View API logs:    tail -f logs/api.log"
echo "  • View worker logs: tail -f logs/worker.log"
echo "  • Stop services:    ./scripts/stop.sh"
echo "  • Restart services: ./scripts/stop.sh && ./scripts/run.sh"

#!/bin/bash

# Blockchain Explorer - Docker Setup Script
# This script helps set up PostgreSQL using Docker Compose

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Get the project root directory
PROJECT_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$PROJECT_ROOT"

echo -e "${GREEN}=== Blockchain Explorer - Docker Setup ===${NC}"
echo ""

# Check if docker and docker-compose are installed
if ! command -v docker &> /dev/null; then
    echo -e "${RED}Error: Docker is not installed${NC}"
    echo "Please install Docker Desktop from: https://www.docker.com/products/docker-desktop"
    exit 1
fi

if ! command -v docker-compose &> /dev/null && ! docker compose version &> /dev/null; then
    echo -e "${RED}Error: Docker Compose is not installed${NC}"
    echo "Please install Docker Compose"
    exit 1
fi

# Check if .env file exists
if [ ! -f .env ]; then
    echo -e "${YELLOW}Creating .env file from .env.example...${NC}"
    cp .env.example .env
    echo -e "${GREEN}✓ .env file created${NC}"
    echo ""
    echo -e "${YELLOW}Please review and update .env file with your settings${NC}"
    echo "Especially update your RPC_URL with a valid API key"
    echo ""
fi

# Load environment variables
export $(grep -v '^#' .env | xargs)

echo "Docker Compose Setup Options:"
echo ""
echo "1) Start PostgreSQL and pgAdmin"
echo "2) Start PostgreSQL only (no pgAdmin)"
echo "3) Stop all services"
echo "4) Stop and remove all data (destructive)"
echo "5) View logs"
echo "6) Database status"
echo "7) Exit"
echo ""
read -p "Select an option [1-7]: " option

case $option in
    1)
        echo -e "${YELLOW}Starting PostgreSQL and pgAdmin...${NC}"
        docker-compose up -d
        echo ""
        echo -e "${GREEN}✓ Services started${NC}"
        echo ""
        echo "PostgreSQL:"
        echo "  Host: localhost"
        echo "  Port: ${DB_PORT:-5432}"
        echo "  Database: ${DB_NAME:-blockchain_explorer}"
        echo "  User: ${DB_USER:-postgres}"
        echo "  Password: ${DB_PASSWORD:-postgres}"
        echo ""
        echo "pgAdmin (Database Management UI):"
        echo "  URL: http://localhost:${PGADMIN_PORT:-5050}"
        echo "  Email: ${PGADMIN_EMAIL:-admin@blockchain-explorer.local}"
        echo "  Password: ${PGADMIN_PASSWORD:-admin}"
        echo ""
        echo -e "${YELLOW}Waiting for PostgreSQL to be ready...${NC}"
        sleep 5

        # Check if PostgreSQL is ready
        if docker exec blockchain-explorer-db pg_isready -U ${DB_USER:-postgres} -d ${DB_NAME:-blockchain_explorer} &> /dev/null; then
            echo -e "${GREEN}✓ PostgreSQL is ready${NC}"
            echo ""
            echo "Next steps:"
            echo "  1. Database migrations will run automatically on first start"
            echo "  2. Run the application: ./scripts/run.sh"
            echo "  3. Access pgAdmin at http://localhost:${PGADMIN_PORT:-5050}"
        else
            echo -e "${YELLOW}PostgreSQL is still starting up. Please wait a moment and check with: docker-compose logs postgres${NC}"
        fi
        ;;

    2)
        echo -e "${YELLOW}Starting PostgreSQL only...${NC}"
        docker-compose up -d postgres
        echo ""
        echo -e "${GREEN}✓ PostgreSQL started${NC}"
        echo ""
        echo "Connection details:"
        echo "  Host: localhost"
        echo "  Port: ${DB_PORT:-5432}"
        echo "  Database: ${DB_NAME:-blockchain_explorer}"
        echo "  User: ${DB_USER:-postgres}"
        echo "  Password: ${DB_PASSWORD:-postgres}"
        ;;

    3)
        echo -e "${YELLOW}Stopping all services...${NC}"
        docker-compose down
        echo -e "${GREEN}✓ Services stopped${NC}"
        echo ""
        echo "Data is preserved in Docker volumes"
        echo "To remove data as well, use option 4"
        ;;

    4)
        echo -e "${RED}⚠️  WARNING: This will delete all database data!${NC}"
        read -p "Are you sure? Type 'yes' to confirm: " confirm
        if [ "$confirm" = "yes" ]; then
            echo -e "${YELLOW}Stopping services and removing data...${NC}"
            docker-compose down -v
            echo -e "${GREEN}✓ Services stopped and data removed${NC}"
        else
            echo "Cancelled"
        fi
        ;;

    5)
        echo -e "${YELLOW}Showing logs (Ctrl+C to exit)...${NC}"
        docker-compose logs -f
        ;;

    6)
        echo -e "${YELLOW}Checking database status...${NC}"
        echo ""

        if docker ps --filter "name=blockchain-explorer-db" --format "{{.Names}}" | grep -q blockchain-explorer-db; then
            echo -e "${GREEN}✓ PostgreSQL is running${NC}"

            # Show container stats
            echo ""
            echo "Container Stats:"
            docker stats --no-stream blockchain-explorer-db

            # Check database connection
            echo ""
            echo "Database Connection:"
            if docker exec blockchain-explorer-db pg_isready -U ${DB_USER:-postgres} -d ${DB_NAME:-blockchain_explorer} &> /dev/null; then
                echo -e "${GREEN}✓ Database is accepting connections${NC}"

                # Show database size
                DB_SIZE=$(docker exec blockchain-explorer-db psql -U ${DB_USER:-postgres} -d ${DB_NAME:-blockchain_explorer} -t -c "SELECT pg_size_pretty(pg_database_size('${DB_NAME:-blockchain_explorer}'));" 2>/dev/null | tr -d ' ')
                echo "Database size: $DB_SIZE"
            else
                echo -e "${RED}✗ Database is not ready${NC}"
            fi
        else
            echo -e "${RED}✗ PostgreSQL is not running${NC}"
            echo "Start it with option 1 or 2"
        fi

        echo ""

        if docker ps --filter "name=blockchain-explorer-pgadmin" --format "{{.Names}}" | grep -q blockchain-explorer-pgadmin; then
            echo -e "${GREEN}✓ pgAdmin is running${NC}"
            echo "  URL: http://localhost:${PGADMIN_PORT:-5050}"
        else
            echo -e "${YELLOW}✗ pgAdmin is not running${NC}"
        fi
        ;;

    7)
        echo "Exiting..."
        exit 0
        ;;

    *)
        echo -e "${RED}Invalid option${NC}"
        exit 1
        ;;
esac

echo ""
echo "Commands:"
echo "  • Run this script again: ./scripts/docker-setup.sh"
echo "  • View logs:            docker-compose logs -f"
echo "  • Stop services:        docker-compose down"
echo "  • Restart services:     docker-compose restart"

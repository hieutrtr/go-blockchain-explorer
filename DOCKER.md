# Docker Setup Guide

This guide explains how to use Docker Compose to run PostgreSQL for the Blockchain Explorer.

## Quick Start

```bash
# Start everything (PostgreSQL + pgAdmin)
./scripts/docker-setup.sh

# Or manually
docker-compose up -d
```

## What's Included

### PostgreSQL 16
- Latest stable PostgreSQL with Alpine Linux
- Automatic database initialization
- Persistent data storage
- Health checks enabled
- Port: 5432 (configurable via `.env`)

### pgAdmin 4 (Optional)
- Web-based PostgreSQL management tool
- Accessible at http://localhost:5050
- Pre-configured for easy setup
- Port: 5050 (configurable via `.env`)

## Configuration

All configuration is done via the `.env` file:

```bash
# Database Configuration
DB_HOST=localhost
DB_PORT=5432
DB_NAME=blockchain_explorer
DB_USER=postgres
DB_PASSWORD=postgres

# pgAdmin Configuration
PGADMIN_EMAIL=admin@blockchain-explorer.local
PGADMIN_PASSWORD=admin
PGADMIN_PORT=5050
```

## Common Commands

### Starting Services

```bash
# Start all services
docker-compose up -d

# Start PostgreSQL only (no pgAdmin)
docker-compose up -d postgres

# Start with logs visible
docker-compose up
```

### Stopping Services

```bash
# Stop all services (keeps data)
docker-compose down

# Stop and remove all data (destructive!)
docker-compose down -v
```

### Viewing Logs

```bash
# All logs
docker-compose logs -f

# PostgreSQL logs only
docker-compose logs -f postgres

# pgAdmin logs only
docker-compose logs -f pgadmin

# Last 100 lines
docker-compose logs --tail=100 postgres
```

### Managing Services

```bash
# Restart services
docker-compose restart

# Restart PostgreSQL only
docker-compose restart postgres

# Check status
docker-compose ps

# View resource usage
docker stats blockchain-explorer-db
```

### Database Operations

```bash
# Connect to PostgreSQL with psql
docker exec -it blockchain-explorer-db psql -U postgres -d blockchain_explorer

# Run SQL file
docker exec -i blockchain-explorer-db psql -U postgres -d blockchain_explorer < migrations/001_initial_schema.sql

# Create database backup
docker exec blockchain-explorer-db pg_dump -U postgres blockchain_explorer > backup.sql

# Restore database backup
docker exec -i blockchain-explorer-db psql -U postgres -d blockchain_explorer < backup.sql

# Get database size
docker exec blockchain-explorer-db psql -U postgres -d blockchain_explorer -c "SELECT pg_size_pretty(pg_database_size('blockchain_explorer'));"
```

### Troubleshooting

```bash
# Check if PostgreSQL is ready
docker exec blockchain-explorer-db pg_isready -U postgres

# View container details
docker inspect blockchain-explorer-db

# Check container health
docker ps --format "table {{.Names}}\t{{.Status}}\t{{.Ports}}"

# Remove everything and start fresh
docker-compose down -v
docker-compose up -d
```

## Using pgAdmin

### Accessing pgAdmin

1. Open browser: http://localhost:5050
2. Login with:
   - Email: admin@blockchain-explorer.local
   - Password: admin

### Connecting to PostgreSQL

1. Right-click "Servers" in left sidebar
2. Select "Register" → "Server"
3. In the "General" tab:
   - Name: Blockchain Explorer
4. In the "Connection" tab:
   - Host name/address: **postgres** (container name)
   - Port: 5432
   - Maintenance database: blockchain_explorer
   - Username: postgres
   - Password: postgres
5. Click "Save"

### Using pgAdmin

- **Query Tool**: Right-click database → "Query Tool"
- **View Tables**: Expand database → Schemas → public → Tables
- **Run Queries**: Write SQL and press F5 or click Execute
- **Export Data**: Right-click table → Import/Export

## Data Persistence

Data is stored in Docker volumes:

```bash
# List volumes
docker volume ls | grep blockchain-explorer

# Inspect volume
docker volume inspect go-blockchain-explorer_postgres_data

# View volume location
docker volume inspect go-blockchain-explorer_postgres_data --format '{{ .Mountpoint }}'

# Remove volumes (WARNING: deletes all data)
docker volume rm go-blockchain-explorer_postgres_data
docker volume rm go-blockchain-explorer_pgadmin_data
```

## Network Configuration

Services communicate via the `blockchain-explorer` Docker network:

```bash
# Inspect network
docker network inspect go-blockchain-explorer_blockchain-explorer

# List containers in network
docker network inspect go-blockchain-explorer_blockchain-explorer --format '{{range .Containers}}{{.Name}} {{end}}'
```

## Production Considerations

### Security

```bash
# Use strong passwords in .env
DB_PASSWORD=your_strong_password_here
PGADMIN_PASSWORD=your_strong_password_here

# Disable pgAdmin in production
docker-compose up -d postgres
```

### Performance

```bash
# Adjust PostgreSQL settings by creating docker-compose.override.yml
services:
  postgres:
    command:
      - "postgres"
      - "-c"
      - "shared_buffers=256MB"
      - "-c"
      - "max_connections=200"
```

### Backups

Set up automated backups:

```bash
# Create backup script
cat > backup.sh << 'EOF'
#!/bin/bash
BACKUP_DIR="backups"
mkdir -p $BACKUP_DIR
DATE=$(date +%Y%m%d_%H%M%S)
docker exec blockchain-explorer-db pg_dump -U postgres blockchain_explorer | gzip > $BACKUP_DIR/backup_$DATE.sql.gz
find $BACKUP_DIR -mtime +7 -delete  # Keep only 7 days
EOF

chmod +x backup.sh

# Add to crontab for daily backups
# 0 2 * * * /path/to/backup.sh
```

## Monitoring

### Health Checks

```bash
# Check health status
docker inspect blockchain-explorer-db --format '{{.State.Health.Status}}'

# View health check logs
docker inspect blockchain-explorer-db --format '{{json .State.Health}}' | jq
```

### Resource Usage

```bash
# Monitor real-time stats
docker stats blockchain-explorer-db

# Get container metrics
docker inspect blockchain-explorer-db --format '{{.State.Status}} {{.State.Running}}'
```

## Advanced Configuration

### Custom PostgreSQL Configuration

Create `postgres.conf` and mount it:

```yaml
# In docker-compose.override.yml
services:
  postgres:
    volumes:
      - ./postgres.conf:/etc/postgresql/postgresql.conf
    command: postgres -c config_file=/etc/postgresql/postgresql.conf
```

### Using Different PostgreSQL Version

```yaml
# In docker-compose.yml
services:
  postgres:
    image: postgres:15-alpine  # or postgres:14-alpine
```

### Resource Limits

```yaml
# In docker-compose.override.yml
services:
  postgres:
    deploy:
      resources:
        limits:
          cpus: '2'
          memory: 2G
        reservations:
          cpus: '1'
          memory: 1G
```

## Troubleshooting

### Port Already in Use

```bash
# Find what's using port 5432
lsof -i :5432

# Change port in .env
DB_PORT=5433

# Restart services
docker-compose down && docker-compose up -d
```

### Container Won't Start

```bash
# Check logs
docker-compose logs postgres

# Remove and recreate
docker-compose down
docker-compose up -d --force-recreate
```

### Permission Issues

```bash
# Fix volume permissions
docker-compose down
docker volume rm go-blockchain-explorer_postgres_data
docker-compose up -d
```

### Connection Refused

```bash
# Wait for PostgreSQL to be ready
docker exec blockchain-explorer-db pg_isready -U postgres

# Check if container is running
docker ps | grep blockchain-explorer-db

# Check network connectivity
docker exec blockchain-explorer-db ping -c 1 postgres
```

## Migration from Local PostgreSQL

If you're switching from local PostgreSQL to Docker:

```bash
# 1. Backup existing database
pg_dump -U postgres blockchain_explorer > backup.sql

# 2. Start Docker PostgreSQL
docker-compose up -d postgres

# 3. Restore to Docker
docker exec -i blockchain-explorer-db psql -U postgres -d blockchain_explorer < backup.sql

# 4. Verify data
docker exec blockchain-explorer-db psql -U postgres -d blockchain_explorer -c "\dt"
```

## Support

For issues or questions:
- Check logs: `docker-compose logs -f`
- Run setup script: `./scripts/docker-setup.sh` (option 6 for status)
- GitHub Issues: [Create an issue](https://github.com/hieutt50/go-blockchain-explorer/issues)

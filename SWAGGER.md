# Swagger / OpenAPI Documentation

Interactive API documentation using Swagger UI.

## Quick Start

### Option 1: Using Docker (Recommended)

```bash
# Start Swagger UI
docker-compose -f docker-compose.swagger.yml up -d

# Access Swagger UI
open http://localhost:8081
```

### Option 2: Using Makefile

```bash
# Start Swagger UI
make swagger-up

# Stop Swagger UI
make swagger-down
```

### Option 3: Online Swagger Editor

1. Go to https://editor.swagger.io/
2. File → Import URL
3. Paste your raw `openapi.yaml` URL from GitHub
4. Or copy/paste the content of `openapi.yaml` directly

## Accessing Swagger UI

Once Swagger UI is running, open your browser:

**URL:** http://localhost:8081

You'll see an interactive API documentation where you can:
- Browse all available endpoints
- View request/response schemas
- Try out API calls directly from the browser
- See example requests and responses

## Using Swagger UI

### 1. Browse Endpoints

Click on any endpoint to expand it and see:
- Description
- Parameters
- Request body schema
- Response codes and schemas
- Examples

### 2. Try It Out

1. Click on an endpoint (e.g., `GET /v1/blocks`)
2. Click the "Try it out" button
3. Enter parameters if required
4. Click "Execute"
5. View the response below

### 3. Example: List Blocks

```
GET /v1/blocks
```

1. Expand the endpoint
2. Click "Try it out"
3. Set parameters:
   - `limit`: 10
   - `offset`: 0
4. Click "Execute"
5. See the response with actual blockchain data

### 4. Example: Get Block by Height

```
GET /v1/blocks/{heightOrHash}
```

1. Expand the endpoint
2. Click "Try it out"
3. Enter a block height (e.g., `18500000`)
4. Click "Execute"
5. See the block details

## OpenAPI Specification

The API is documented using OpenAPI 3.0.3 specification.

**File:** `openapi.yaml`

This file contains:
- All endpoint definitions
- Request/response schemas
- Parameter descriptions
- Example values
- Error responses

## Endpoints Summary

### Health & Monitoring
- `GET /health` - Health check
- `GET /metrics` - Prometheus metrics

### Blocks
- `GET /v1/blocks` - List recent blocks (paginated)
- `GET /v1/blocks/{heightOrHash}` - Get block by height or hash

### Transactions
- `GET /v1/txs/{hash}` - Get transaction by hash

### Addresses
- `GET /v1/address/{addr}/txs` - Get address transaction history (paginated)

### Event Logs
- `GET /v1/logs` - Query event logs with filters (paginated)

### Statistics
- `GET /v1/stats/chain` - Get chain statistics

## Configuration

### Change Swagger UI Port

Edit `docker-compose.swagger.yml`:

```yaml
ports:
  - "8082:8080"  # Change 8081 to your preferred port
```

### Update API Base URL

If your API is running on a different host/port, update `openapi.yaml`:

```yaml
servers:
  - url: http://your-api-host:8080
    description: Your API server
```

## Updating Documentation

When you add or modify API endpoints:

1. Update `openapi.yaml` with the new endpoint definition
2. Restart Swagger UI:
   ```bash
   docker-compose -f docker-compose.swagger.yml restart
   ```
3. Refresh your browser

## Alternative Tools

### 1. Redoc

For a different documentation UI:

```bash
docker run -p 8082:80 \
  -e SPEC_URL=http://host.docker.internal:8081/openapi.yaml \
  redocly/redoc
```

### 2. Swagger Editor

For editing the OpenAPI spec:

```bash
docker run -p 8083:8080 swaggerapi/swagger-editor
```

Then import `openapi.yaml` via File → Import URL

### 3. OpenAPI Generator

Generate client SDKs from the OpenAPI spec:

```bash
# Install openapi-generator
npm install -g @openapitools/openapi-generator-cli

# Generate TypeScript client
openapi-generator-cli generate \
  -i openapi.yaml \
  -g typescript-fetch \
  -o ./clients/typescript

# Generate Python client
openapi-generator-cli generate \
  -i openapi.yaml \
  -g python \
  -o ./clients/python

# Generate Go client
openapi-generator-cli generate \
  -i openapi.yaml \
  -g go \
  -o ./clients/go
```

## Validation

Validate your OpenAPI specification:

```bash
# Using docker
docker run --rm -v ${PWD}:/local openapitools/openapi-generator-cli validate \
  -i /local/openapi.yaml

# Using npx
npx @openapitools/openapi-generator-cli validate -i openapi.yaml
```

## Export Documentation

### Generate Static HTML

```bash
docker run --rm \
  -v ${PWD}:/local \
  openapitools/openapi-generator-cli generate \
  -i /local/openapi.yaml \
  -g html2 \
  -o /local/docs/api
```

### Generate Markdown

```bash
docker run --rm \
  -v ${PWD}:/local \
  openapitools/openapi-generator-cli generate \
  -i /local/openapi.yaml \
  -g markdown \
  -o /local/docs/api-markdown
```

## Troubleshooting

### Swagger UI not loading

```bash
# Check if container is running
docker ps | grep swagger

# Check logs
docker-compose -f docker-compose.swagger.yml logs swagger-ui

# Restart
docker-compose -f docker-compose.swagger.yml restart
```

### API calls failing from Swagger UI

If you get CORS errors when testing from Swagger UI:

1. Make sure the API server is running: `curl http://localhost:8080/health`
2. Check CORS settings in your `.env`:
   ```bash
   API_CORS_ORIGINS=*
   ```
3. Restart API server: `make stop && make run`

### Port already in use

```bash
# Check what's using port 8081
lsof -i :8081

# Change port in docker-compose.swagger.yml
ports:
  - "8082:8080"
```

## Integration with CI/CD

### Validate on PR

Add to your `.github/workflows/ci.yml`:

```yaml
- name: Validate OpenAPI Spec
  run: |
    docker run --rm -v ${PWD}:/local \
      openapitools/openapi-generator-cli validate \
      -i /local/openapi.yaml
```

### Generate Client SDKs Automatically

```yaml
- name: Generate Clients
  run: |
    npm install -g @openapitools/openapi-generator-cli
    openapi-generator-cli generate -i openapi.yaml -g typescript-fetch -o ./clients/typescript
    openapi-generator-cli generate -i openapi.yaml -g python -o ./clients/python
```

## Best Practices

1. **Keep openapi.yaml in sync** - Update it whenever you change API endpoints
2. **Use examples** - Add example values for all parameters and responses
3. **Document errors** - Include all possible error responses
4. **Version your API** - Use `/v1`, `/v2` prefixes for API versioning
5. **Test via Swagger UI** - Use "Try it out" to verify all endpoints work

## Resources

- **OpenAPI Specification**: https://swagger.io/specification/
- **Swagger UI**: https://swagger.io/tools/swagger-ui/
- **OpenAPI Generator**: https://openapi-generator.tech/
- **Swagger Editor**: https://editor.swagger.io/

## Support

For issues or questions:
- [GitHub Issues](https://github.com/hieutt50/go-blockchain-explorer/issues)
- [API Documentation](API.md)
- [Main README](README.md)

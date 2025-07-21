# Health Check Feature

This package provides health check functionality for the mcp-grafana server, enabling automated monitoring and healing of the service.

## Overview

The health check feature adds HTTP endpoints that can be used by load balancers, orchestrators (like Kubernetes), and monitoring systems to determine if the mcp-grafana server is running and healthy.

## Features

- Multiple health check endpoints (`/healthz`, `/health`, `/health/readiness`, `/health/liveness`)
- Configurable to run on the same port as the main server or a separate port
- JSON response format with service information
- Simple text response option for basic health checks
- Graceful shutdown support
- Thread-safe concurrent request handling

## Available Endpoints

### `/healthz` and `/health`
Returns detailed health information in JSON format:
```json
{
  "status": "healthy",
  "service": "mcp-grafana", 
  "version": "v1.0.0",
  "timestamp": "2024-01-01T12:00:00Z"
}
```

### `/health/readiness`
Same as `/health` - indicates the service is ready to serve requests.

### `/health/liveness`
Returns a simple "OK" text response to indicate the service is alive.

## Configuration

Health checks are enabled by default for `sse` and `streamable-http` transports. The `stdio` transport does not support health checks as it's used for direct communication.

### Command Line Flags

- `--health-enabled`: Enable/disable health check endpoints (default: true)
- `--health-port`: Specific port/address for health checks (optional)
- `--health-separate-port`: Run health checks on separate port (default: true)

### Examples

#### Enable health checks on separate port (default behavior)
```bash
./mcp-grafana --transport sse --address localhost:8000
# Main server: localhost:8000
# Health checks: localhost:9000 (automatically assigned)
```

#### Custom health check port
```bash
./mcp-grafana --transport sse --address localhost:8000 --health-port localhost:8080
# Main server: localhost:8000  
# Health checks: localhost:8080
```

#### Health checks on same port as main server
```bash
./mcp-grafana --transport sse --address localhost:8000 --health-separate-port=false
# Main server and health checks: localhost:8000
```

#### Disable health checks
```bash
./mcp-grafana --transport sse --address localhost:8000 --health-enabled=false
# Only main server: localhost:8000
# No health check endpoints
```

## Use Cases

### Kubernetes Deployments
```yaml
apiVersion: apps/v1
kind: Deployment
spec:
  template:
    spec:
      containers:
      - name: mcp-grafana
        image: mcp-grafana:latest
        ports:
        - containerPort: 8000
        - containerPort: 9000
        livenessProbe:
          httpGet:
            path: /health/liveness
            port: 9000
          initialDelaySeconds: 30
          periodSeconds: 10
        readinessProbe:
          httpGet:
            path: /health/readiness
            port: 9000
          initialDelaySeconds: 5
          periodSeconds: 5
```

### Docker Compose with Health Checks
```yaml
version: '3.8'
services:
  mcp-grafana:
    image: mcp-grafana:latest
    command: ["./mcp-grafana", "--transport", "sse", "--address", "0.0.0.0:8000"]
    ports:
      - "8000:8000"
      - "9000:9000"
    healthcheck:
      test: ["CMD", "curl", "-f", "http://localhost:9000/healthz"]
      interval: 30s
      timeout: 10s
      retries: 3
      start_period: 40s
```

### Load Balancer Configuration
```nginx
upstream mcp_grafana {
    server 10.0.0.1:8000;
    server 10.0.0.2:8000;
    server 10.0.0.3:8000;
}

server {
    location /healthz {
        proxy_pass http://10.0.0.1:9000/healthz;
    }
    
    location / {
        proxy_pass http://mcp_grafana;
    }
}
```

## HTTP Status Codes

- `200 OK`: Service is healthy and ready
- `405 Method Not Allowed`: Only GET requests are supported
- `500 Internal Server Error`: Error occurred while generating response

## Implementation Details

The health check server:
- Runs independently of the main MCP server
- Supports graceful shutdown with configurable timeout
- Uses minimal resources (separate lightweight HTTP server)
- Thread-safe for concurrent requests
- Automatically selects available ports when needed

## Security Considerations

- Health check endpoints return minimal information
- No authentication required (by design for monitoring systems)
- Consider network security (firewall rules) if exposing health ports
- Health checks run on separate port by default to isolate from main service

## Troubleshooting

### Health check port conflicts
If you encounter port binding errors, either:
1. Specify a custom port with `--health-port`
2. Let the system auto-assign with the default behavior
3. Use `--health-separate-port=false` to share the main server port

### Health checks not accessible
1. Verify the server transport supports health checks (`sse` or `streamable-http`)
2. Check that `--health-enabled=true` (default)
3. Ensure firewall allows access to the health check port
4. Check server logs for health server startup messages
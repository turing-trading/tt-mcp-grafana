# Health Check Feature

This package provides a simple health check endpoint for the mcp-grafana server, enabling automated monitoring and healing of the service.

## Overview

The health check feature adds a `/healthz` HTTP endpoint that can be used by load balancers, orchestrators (like Kubernetes), and monitoring systems to determine if the mcp-grafana server is running and healthy.

## Features

- Single `/healthz` health check endpoint
- Simple "OK" text response for easy parsing
- Runs on the same port as the main server
- Supported for `sse` and `streamable-http` transports only
- Minimal overhead and complexity

## Available Endpoint

### `/healthz`
Returns a simple "OK" text response with HTTP 200 status:
```
OK
```

Only GET requests are supported. Other HTTP methods return `405 Method Not Allowed`.

## Configuration

Health checks are enabled by default for `sse` and `streamable-http` transports. The `stdio` transport does not support health checks as it's used for direct communication.

### Command Line Flag

- `--health-enabled`: Enable/disable health check endpoint (default: true)

### Examples

#### Enable health checks (default behavior)
```bash
./mcp-grafana --transport sse --address localhost:8000
# Server with /healthz endpoint: localhost:8000
```

#### Disable health checks
```bash
./mcp-grafana --transport sse --address localhost:8000 --health-enabled=false
# Server without health checks: localhost:8000
```

#### Health checks only work with server transports
```bash
./mcp-grafana --transport stdio  # No health checks (stdio doesn't support them)
./mcp-grafana --transport sse --address localhost:8000  # Health checks enabled
./mcp-grafana --transport streamable-http --address localhost:8000  # Health checks enabled
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
        livenessProbe:
          httpGet:
            path: /healthz
            port: 8000
          initialDelaySeconds: 30
          periodSeconds: 10
        readinessProbe:
          httpGet:
            path: /healthz
            port: 8000
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
    healthcheck:
      test: ["CMD", "curl", "-f", "http://localhost:8000/healthz"]
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
        access_log off;
        return 200 "healthy\n";
        add_header Content-Type text/plain;
    }
    
    location / {
        proxy_pass http://mcp_grafana;
        
        # Health check for upstream servers
        proxy_next_upstream error timeout http_500 http_502 http_503 http_504;
    }
}
```

## HTTP Status Codes

- `200 OK`: Service is healthy and ready
- `405 Method Not Allowed`: Only GET requests are supported

## Implementation Details

The health check implementation:
- Uses a reverse proxy approach: MCP server runs on internal port, public server adds `/healthz`
- Minimal performance impact
- Thread-safe for concurrent requests
- No external dependencies
- Simple text response for maximum compatibility

## Transport Support

| Transport | Health Check Support | Notes |
|-----------|---------------------|--------|
| `stdio` | ❌ No | Direct communication, no HTTP server |
| `sse` | ✅ Yes | Server-Sent Events HTTP server |
| `streamable-http` | ✅ Yes | HTTP-based MCP server |

## Security Considerations

- Health check endpoint returns minimal information (just "OK")
- No authentication required (by design for monitoring systems)
- Consider network security (firewall rules) if exposing publicly
- Health checks share the same port as the main service

## Troubleshooting

### Health check not accessible
1. Verify the server transport supports health checks (`sse` or `streamable-http`)
2. Check that `--health-enabled=true` (default)
3. Ensure server is running and listening on the expected port
4. Check firewall allows access to the server port
5. Verify using curl: `curl http://localhost:8000/healthz`

### Expected responses
- **Healthy server**: `HTTP 200` with body `OK`
- **Wrong method**: `HTTP 405` with body `Method not allowed`
- **Server down**: Connection refused or timeout error
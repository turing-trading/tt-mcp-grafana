package health

import (
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/mark3labs/mcp-go/server"
)

// StartSSEWithHealth starts an SSE server with /healthz endpoint on the same port
func StartSSEWithHealth(mcpServer *server.MCPServer, addr string, contextFunc server.SSEContextFunc, basePath string) error {
	// Get internal address (increment port by 1)
	internalAddr, err := getInternalAddr(addr)
	if err != nil {
		return fmt.Errorf("failed to get internal address: %w", err)
	}

	// Start MCP server on internal port
	go func() {
		srv := server.NewSSEServer(
			mcpServer,
			server.WithSSEContextFunc(contextFunc),
			server.WithStaticBasePath(basePath),
		)
		slog.Info("Starting internal MCP SSE server", "address", internalAddr)
		if err := srv.Start(internalAddr); err != nil {
			slog.Error("Internal MCP SSE server failed", "error", err)
		}
	}()

	// Wait a moment for the internal server to start
	time.Sleep(100 * time.Millisecond)

	// Start public server with health check
	return startPublicServer(addr, internalAddr)
}

// StartStreamableHTTPWithHealth starts a StreamableHTTP server with /healthz endpoint on the same port
func StartStreamableHTTPWithHealth(mcpServer *server.MCPServer, addr string, contextFunc server.HTTPContextFunc, endpointPath string) error {
	// Get internal address (increment port by 1)
	internalAddr, err := getInternalAddr(addr)
	if err != nil {
		return fmt.Errorf("failed to get internal address: %w", err)
	}

	// Start MCP server on internal port
	go func() {
		srv := server.NewStreamableHTTPServer(
			mcpServer,
			server.WithHTTPContextFunc(contextFunc),
			server.WithStateLess(true),
			server.WithEndpointPath(endpointPath),
		)
		slog.Info("Starting internal MCP StreamableHTTP server", "address", internalAddr)
		if err := srv.Start(internalAddr); err != nil {
			slog.Error("Internal MCP StreamableHTTP server failed", "error", err)
		}
	}()

	// Wait a moment for the internal server to start
	time.Sleep(100 * time.Millisecond)

	// Start public server with health check
	return startPublicServer(addr, internalAddr)
}

// startPublicServer starts a server that handles /healthz and proxies other requests
func startPublicServer(publicAddr, internalAddr string) error {
	// Create proxy to internal MCP server
	target, err := url.Parse(fmt.Sprintf("http://%s", internalAddr))
	if err != nil {
		return fmt.Errorf("invalid internal address: %w", err)
	}

	proxy := httputil.NewSingleHostReverseProxy(target)

	// Create handler
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Handle health check
		if r.URL.Path == "/healthz" {
			if r.Method != http.MethodGet {
				http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
				return
			}
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("OK"))
			return
		}

		// Proxy everything else to internal MCP server
		proxy.ServeHTTP(w, r)
	})

	// Start public server
	publicServer := &http.Server{
		Addr:    publicAddr,
		Handler: handler,
	}

	slog.Info("Starting Grafana MCP server with /healthz health check", "address", publicAddr)
	return publicServer.ListenAndServe()
}

// getInternalAddr creates an internal address by incrementing the port
func getInternalAddr(publicAddr string) (string, error) {
	// Split host and port
	parts := strings.Split(publicAddr, ":")
	if len(parts) != 2 {
		return "", fmt.Errorf("invalid address format: %s", publicAddr)
	}

	host := parts[0]
	portStr := parts[1]

	// Parse port number
	port, err := strconv.Atoi(portStr)
	if err != nil {
		return "", fmt.Errorf("invalid port number: %s", portStr)
	}

	// Increment port for internal server
	internalPort := port + 1
	return fmt.Sprintf("%s:%d", host, internalPort), nil
}

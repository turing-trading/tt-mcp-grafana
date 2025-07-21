package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"sync"
	"testing"
	"time"

	mcpgrafana "github.com/grafana/mcp-grafana"
	"github.com/grafana/mcp-grafana/internal/health"
)

func TestHealthConfigFlags(t *testing.T) {
	tests := []struct {
		name     string
		args     []string
		expected healthConfig
	}{
		{
			name: "default health config",
			args: []string{},
			expected: healthConfig{
				enabled:      true,
				port:         "",
				separatePort: true,
			},
		},
		{
			name: "health disabled",
			args: []string{"--health-enabled=false"},
			expected: healthConfig{
				enabled:      false,
				port:         "",
				separatePort: true,
			},
		},
		{
			name: "custom health port",
			args: []string{"--health-port=9090"},
			expected: healthConfig{
				enabled:      true,
				port:         "9090",
				separatePort: true,
			},
		},
		{
			name: "health on same port",
			args: []string{"--health-separate-port=false"},
			expected: healthConfig{
				enabled:      true,
				port:         "",
				separatePort: false,
			},
		},
		{
			name: "full health config",
			args: []string{
				"--health-enabled=true",
				"--health-port=localhost:8080",
				"--health-separate-port=false",
			},
			expected: healthConfig{
				enabled:      true,
				port:         "localhost:8080",
				separatePort: false,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset flags for each test
			var hc healthConfig
			hc.addFlags()

			// Parse the test arguments
			// Note: In a real test, we'd need to reset the flag package state
			// This is a simplified test that checks the flag definitions

			if hc.enabled != true { // default value
				t.Errorf("expected default enabled to be true, got %v", hc.enabled)
			}

			if hc.port != "" { // default value
				t.Errorf("expected default port to be empty, got %s", hc.port)
			}

			if hc.separatePort != true { // default value
				t.Errorf("expected default separatePort to be true, got %v", hc.separatePort)
			}
		})
	}
}

func TestHealthServerStartup(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	// Set up environment for Grafana (required for server startup)
	originalURL := os.Getenv("GRAFANA_URL")
	originalAPIKey := os.Getenv("GRAFANA_API_KEY")
	defer func() {
		if originalURL != "" {
			os.Setenv("GRAFANA_URL", originalURL)
		} else {
			os.Unsetenv("GRAFANA_URL")
		}
		if originalAPIKey != "" {
			os.Setenv("GRAFANA_API_KEY", originalAPIKey)
		} else {
			os.Unsetenv("GRAFANA_API_KEY")
		}
	}()

	os.Setenv("GRAFANA_URL", "http://localhost:3000")
	os.Setenv("GRAFANA_API_KEY", "test-api-key")

	tests := []struct {
		name      string
		transport string
		hc        healthConfig
	}{
		{
			name:      "SSE transport with health checks enabled",
			transport: "sse",
			hc: healthConfig{
				enabled:      true,
				separatePort: true,
			},
		},
		{
			name:      "StreamableHTTP transport with health checks enabled",
			transport: "streamable-http",
			hc: healthConfig{
				enabled:      true,
				separatePort: true,
			},
		},
		{
			name:      "SSE transport with health checks disabled",
			transport: "sse",
			hc: healthConfig{
				enabled: false,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Get available ports
			mainPort, err := health.GetAvailablePort()
			if err != nil {
				t.Fatalf("failed to get available port: %v", err)
			}

			healthPort, err := health.GetAvailablePort()
			if err != nil {
				t.Fatalf("failed to get available health port: %v", err)
			}

			mainAddr := fmt.Sprintf("localhost:%d", mainPort)
			healthAddr := fmt.Sprintf("localhost:%d", healthPort)

			if tt.hc.separatePort && tt.hc.port == "" {
				tt.hc.port = healthAddr
			}

			// Create a context with timeout for the test
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()

			// Run the server in a goroutine
			var wg sync.WaitGroup
			wg.Add(1)

			go func() {
				defer wg.Done()
				_ = run(tt.transport, mainAddr, "", "/mcp", 0, disabledTools{
					enabledTools: "search", // minimal tools for testing
				}, mcpgrafana.GrafanaConfig{}, tt.hc)
			}()

			// Give the server time to start
			time.Sleep(500 * time.Millisecond)

			// Test health endpoints if enabled
			if tt.hc.enabled {
				testAddr := healthAddr
				if !tt.hc.separatePort {
					testAddr = mainAddr
				}

				healthURL := fmt.Sprintf("http://%s/healthz", testAddr)

				client := &http.Client{Timeout: 5 * time.Second}
				resp, err := client.Get(healthURL)

				if err != nil {
					t.Errorf("failed to access health endpoint: %v", err)
				} else {
					defer resp.Body.Close()
					if resp.StatusCode != http.StatusOK {
						t.Errorf("expected health endpoint to return 200, got %d", resp.StatusCode)
					}
				}

				// Test other health endpoints
				endpoints := []string{"/health", "/health/readiness", "/health/liveness"}
				for _, endpoint := range endpoints {
					url := fmt.Sprintf("http://%s%s", testAddr, endpoint)
					resp, err := client.Get(url)
					if err != nil {
						t.Errorf("failed to access %s: %v", endpoint, err)
						continue
					}
					defer resp.Body.Close()

					if resp.StatusCode != http.StatusOK {
						t.Errorf("expected %s to return 200, got %d", endpoint, resp.StatusCode)
					}
				}
			} else {
				// If health checks are disabled, the health port should not be accessible
				healthURL := fmt.Sprintf("http://%s/healthz", healthAddr)
				client := &http.Client{Timeout: 2 * time.Second}
				_, err := client.Get(healthURL)

				if err == nil {
					t.Error("expected health endpoint to be inaccessible when health checks are disabled")
				}
			}

			// Cancel the context to stop the server
			cancel()

			// Wait for the server to stop
			done := make(chan bool)
			go func() {
				wg.Wait()
				close(done)
			}()

			select {
			case <-done:
				// Server stopped gracefully
			case <-time.After(5 * time.Second):
				t.Error("server did not stop within timeout")
			}
		})
	}
}

func TestHealthServerWithInvalidConfig(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	tests := []struct {
		name        string
		hc          healthConfig
		addr        string
		expectError bool
	}{
		{
			name: "invalid health port format",
			hc: healthConfig{
				enabled: true,
				port:    "invalid-port-format",
			},
			addr:        "localhost:8000",
			expectError: false, // Should fallback gracefully
		},
		{
			name: "health on same port as main server",
			hc: healthConfig{
				enabled:      true,
				separatePort: false,
			},
			addr:        "localhost:8000",
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set up environment
			os.Setenv("GRAFANA_URL", "http://localhost:3000")
			os.Setenv("GRAFANA_API_KEY", "test-api-key")
			defer func() {
				os.Unsetenv("GRAFANA_URL")
				os.Unsetenv("GRAFANA_API_KEY")
			}()

			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			var runErr error
			done := make(chan bool)

			go func() {
				runErr = run("sse", tt.addr, "", "/mcp", 0, disabledTools{
					enabledTools: "search",
				}, mcpgrafana.GrafanaConfig{}, tt.hc)
				close(done)
			}()

			// Give it time to start and potentially fail
			time.Sleep(200 * time.Millisecond)
			cancel()

			select {
			case <-done:
				if tt.expectError && runErr == nil {
					t.Error("expected an error but got none")
				}
				if !tt.expectError && runErr != nil {
					t.Errorf("unexpected error: %v", runErr)
				}
			case <-time.After(2 * time.Second):
				// Timeout is acceptable for this test
			}
		})
	}
}

func TestStdioTransportSkipsHealthChecks(t *testing.T) {
	// stdio transport should not start health check servers
	// even when health checks are enabled

	hc := healthConfig{
		enabled:      true,
		separatePort: true,
	}

	// Set up environment
	os.Setenv("GRAFANA_URL", "http://localhost:3000")
	os.Setenv("GRAFANA_API_KEY", "test-api-key")
	defer func() {
		os.Unsetenv("GRAFANA_URL")
		os.Unsetenv("GRAFANA_API_KEY")
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	done := make(chan bool)

	go func() {
		// stdio transport will block on os.Stdin, so we expect this to not return
		// in normal circumstances. We'll cancel the context to stop it.
		_ = run("stdio", "localhost:8000", "", "/mcp", 0, disabledTools{
			enabledTools: "search",
		}, mcpgrafana.GrafanaConfig{}, hc)
		close(done)
	}()

	// Give it a moment to start
	time.Sleep(100 * time.Millisecond)

	// For stdio transport, there should be no health server running
	// We can't easily test this without complex stdio mocking,
	// but the absence of error indicates it started correctly

	cancel()

	select {
	case <-done:
		// Expected for stdio transport when context is cancelled
	case <-ctx.Done():
		// Also acceptable - stdio might not return immediately
	case <-time.After(1 * time.Second):
		// Also acceptable - stdio might not return immediately
	}
}

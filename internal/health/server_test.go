package health

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"testing"
	"time"
)

func TestNewServer(t *testing.T) {
	config := Config{
		ServiceName: "test-service",
		Version:     "1.0.0",
	}

	server := NewServer(config)

	if server == nil {
		t.Fatal("NewServer returned nil")
	}

	if server.config.ServiceName != config.ServiceName {
		t.Errorf("expected service name %s, got %s", config.ServiceName, server.config.ServiceName)
	}

	if server.config.Version != config.Version {
		t.Errorf("expected version %s, got %s", config.Version, server.config.Version)
	}

	if server.started {
		t.Error("server should not be started initially")
	}
}

func TestServerStartStop(t *testing.T) {
	config := Config{
		ServiceName: "test-service",
		Version:     "1.0.0",
	}

	server := NewServer(config)

	// Test starting server asynchronously
	port, err := GetAvailablePort()
	if err != nil {
		t.Fatalf("failed to get available port: %v", err)
	}
	addr := fmt.Sprintf("localhost:%d", port)

	err = server.StartAsync(addr)
	if err != nil {
		t.Fatalf("failed to start server: %v", err)
	}

	// Give the server a moment to start
	time.Sleep(100 * time.Millisecond)

	if !server.IsStarted() {
		t.Error("server should be started")
	}

	// Test health endpoints
	baseURL := fmt.Sprintf("http://%s", addr)
	testEndpoints := []string{
		"/healthz",
		"/health",
		"/health/readiness",
		"/health/liveness",
	}

	for _, endpoint := range testEndpoints {
		resp, err := http.Get(baseURL + endpoint)
		if err != nil {
			t.Errorf("failed to GET %s: %v", endpoint, err)
			continue
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Errorf("expected status 200 for %s, got %d", endpoint, resp.StatusCode)
		}
	}

	// Test stopping server
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err = server.Stop(ctx)
	if err != nil {
		t.Errorf("failed to stop server: %v", err)
	}

	if server.IsStarted() {
		t.Error("server should be stopped")
	}

	// Test that endpoints are no longer accessible
	_, err = http.Get(baseURL + "/healthz")
	if err == nil {
		t.Error("expected error when accessing stopped server")
	}
}

func TestServerDoubleStart(t *testing.T) {
	config := Config{
		ServiceName: "test-service",
		Version:     "1.0.0",
	}

	server := NewServer(config)

	port, err := GetAvailablePort()
	if err != nil {
		t.Fatalf("failed to get available port: %v", err)
	}
	addr := fmt.Sprintf("localhost:%d", port)

	// Start server first time
	err = server.StartAsync(addr)
	if err != nil {
		t.Fatalf("failed to start server first time: %v", err)
	}

	// Give the server a moment to start
	time.Sleep(100 * time.Millisecond)

	// Try to start again - should fail
	err = server.StartAsync(addr)
	if err == nil {
		t.Error("expected error when starting already started server")
	}

	// Clean up
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	server.Stop(ctx)
}

func TestServerStopNotStarted(t *testing.T) {
	config := Config{
		ServiceName: "test-service",
		Version:     "1.0.0",
	}

	server := NewServer(config)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Should not error when stopping a server that wasn't started
	err := server.Stop(ctx)
	if err != nil {
		t.Errorf("unexpected error stopping non-started server: %v", err)
	}
}

func TestGetAvailablePort(t *testing.T) {
	port, err := GetAvailablePort()
	if err != nil {
		t.Fatalf("GetAvailablePort failed: %v", err)
	}

	if port <= 0 || port > 65535 {
		t.Errorf("invalid port number: %d", port)
	}

	// Test that we can actually bind to this port
	addr := fmt.Sprintf("localhost:%d", port)

	// Try to listen on the port briefly to verify it's available
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		if contains(err.Error(), "address already in use") {
			t.Errorf("port %d is already in use", port)
		}
	} else {
		listener.Close()
	}
}

func TestGetHealthPort(t *testing.T) {
	tests := []struct {
		name        string
		mainAddr    string
		expected    string
		shouldError bool
	}{
		{
			name:        "valid address",
			mainAddr:    "localhost:8000",
			expected:    "localhost:9000",
			shouldError: false,
		},
		{
			name:        "with IP address",
			mainAddr:    "127.0.0.1:3000",
			expected:    "127.0.0.1:4000",
			shouldError: false,
		},
		{
			name:        "invalid address format",
			mainAddr:    "invalid-address",
			expected:    "",
			shouldError: true,
		},
		{
			name:        "non-numeric port",
			mainAddr:    "localhost:abc",
			expected:    "",
			shouldError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := GetHealthPort(tt.mainAddr)

			if tt.shouldError {
				if err == nil {
					t.Error("expected an error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				if result != tt.expected {
					t.Errorf("expected %s, got %s", tt.expected, result)
				}
			}
		})
	}
}

func TestGenerateHealthAddr(t *testing.T) {
	tests := []struct {
		name     string
		mainAddr string
	}{
		{
			name:     "valid address",
			mainAddr: "localhost:8000",
		},
		{
			name:     "IP address",
			mainAddr: "127.0.0.1:3000",
		},
		{
			name:     "invalid address",
			mainAddr: "invalid-address",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GenerateHealthAddr(tt.mainAddr)

			// Should always return a valid address string
			if result == "" {
				t.Error("GenerateHealthAddr returned empty string")
			}

			// Try to parse the result as a URL to verify it's valid
			if _, err := url.Parse("http://" + result); err != nil {
				t.Errorf("GenerateHealthAddr returned invalid address format: %s", result)
			}
		})
	}
}

func TestServerHealthEndpoints(t *testing.T) {
	config := Config{
		ServiceName: "mcp-grafana",
		Version:     "test-version",
	}

	server := NewServer(config)

	port, err := GetAvailablePort()
	if err != nil {
		t.Fatalf("failed to get available port: %v", err)
	}
	addr := fmt.Sprintf("localhost:%d", port)

	err = server.StartAsync(addr)
	if err != nil {
		t.Fatalf("failed to start server: %v", err)
	}

	// Give the server a moment to start
	time.Sleep(100 * time.Millisecond)

	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		server.Stop(ctx)
	}()

	baseURL := fmt.Sprintf("http://%s", addr)

	// Test different HTTP methods on health endpoints
	tests := []struct {
		endpoint       string
		method         string
		expectedStatus int
	}{
		{"/healthz", http.MethodGet, http.StatusOK},
		{"/healthz", http.MethodPost, http.StatusMethodNotAllowed},
		{"/health", http.MethodGet, http.StatusOK},
		{"/health", http.MethodPut, http.StatusMethodNotAllowed},
		{"/health/readiness", http.MethodGet, http.StatusOK},
		{"/health/readiness", http.MethodDelete, http.StatusMethodNotAllowed},
		{"/health/liveness", http.MethodGet, http.StatusOK},
		{"/health/liveness", http.MethodPost, http.StatusMethodNotAllowed},
	}

	client := &http.Client{Timeout: 5 * time.Second}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("%s %s", tt.method, tt.endpoint), func(t *testing.T) {
			req, err := http.NewRequest(tt.method, baseURL+tt.endpoint, nil)
			if err != nil {
				t.Fatalf("failed to create request: %v", err)
			}

			resp, err := client.Do(req)
			if err != nil {
				t.Fatalf("failed to make request: %v", err)
			}
			defer resp.Body.Close()

			if resp.StatusCode != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, resp.StatusCode)
			}
		})
	}
}

func TestServerConcurrentRequests(t *testing.T) {
	config := Config{
		ServiceName: "test-service",
		Version:     "1.0.0",
	}

	server := NewServer(config)

	port, err := GetAvailablePort()
	if err != nil {
		t.Fatalf("failed to get available port: %v", err)
	}
	addr := fmt.Sprintf("localhost:%d", port)

	err = server.StartAsync(addr)
	if err != nil {
		t.Fatalf("failed to start server: %v", err)
	}

	// Give the server a moment to start
	time.Sleep(100 * time.Millisecond)

	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		server.Stop(ctx)
	}()

	baseURL := fmt.Sprintf("http://%s/healthz", addr)
	client := &http.Client{Timeout: 5 * time.Second}

	// Make concurrent requests
	numRequests := 10
	results := make(chan error, numRequests)

	for i := 0; i < numRequests; i++ {
		go func() {
			resp, err := client.Get(baseURL)
			if err != nil {
				results <- err
				return
			}
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusOK {
				results <- fmt.Errorf("expected status 200, got %d", resp.StatusCode)
				return
			}

			results <- nil
		}()
	}

	// Collect results
	for i := 0; i < numRequests; i++ {
		if err := <-results; err != nil {
			t.Errorf("concurrent request failed: %v", err)
		}
	}
}

// Helper function to check if a string contains a substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr ||
		(len(s) > len(substr) && (s[:len(substr)] == substr || s[len(s)-len(substr):] == substr ||
			containsAt(s, substr))))
}

func containsAt(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

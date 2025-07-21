package health

import (
	"fmt"
	"net/http"
	"testing"
)

func TestGetInternalAddr(t *testing.T) {
	tests := []struct {
		name        string
		publicAddr  string
		expected    string
		shouldError bool
	}{
		{
			name:       "valid localhost address",
			publicAddr: "localhost:8000",
			expected:   "localhost:8001",
		},
		{
			name:       "valid IP address",
			publicAddr: "127.0.0.1:3000",
			expected:   "127.0.0.1:3001",
		},
		{
			name:       "valid hostname with port",
			publicAddr: "example.com:9090",
			expected:   "example.com:9091",
		},
		{
			name:        "invalid address format - no port",
			publicAddr:  "localhost",
			shouldError: true,
		},
		{
			name:        "invalid address format - multiple colons",
			publicAddr:  "host:port:extra",
			shouldError: true,
		},
		{
			name:        "invalid port number",
			publicAddr:  "localhost:abc",
			shouldError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := getInternalAddr(tt.publicAddr)

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

func TestStartPublicServer(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	// This is a basic test that verifies the public server can be created
	// Full integration testing would require starting actual MCP servers
	t.Run("address parsing", func(t *testing.T) {
		// Test that we can generate valid internal addresses
		testAddresses := []string{
			"localhost:8000",
			"127.0.0.1:9000",
			"0.0.0.0:3000",
		}

		for _, addr := range testAddresses {
			internal, err := getInternalAddr(addr)
			if err != nil {
				t.Errorf("failed to generate internal address for %s: %v", addr, err)
			}
			if internal == addr {
				t.Errorf("internal address should be different from public address: %s", addr)
			}
		}
	})
}

func TestHealthEndpointInProxy(t *testing.T) {
	// Test the health endpoint behavior in isolation
	t.Run("health endpoint handler", func(t *testing.T) {
		// Create a mock handler that behaves like our proxy
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/healthz" {
				if r.Method != http.MethodGet {
					http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
					return
				}
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte("OK"))
				return
			}
			// For non-health requests, return a different response
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write([]byte("Not Found"))
		})

		tests := []struct {
			name           string
			path           string
			method         string
			expectedStatus int
			expectedBody   string
		}{
			{
				name:           "GET /healthz returns OK",
				path:           "/healthz",
				method:         http.MethodGet,
				expectedStatus: http.StatusOK,
				expectedBody:   "OK",
			},
			{
				name:           "POST /healthz returns method not allowed",
				path:           "/healthz",
				method:         http.MethodPost,
				expectedStatus: http.StatusMethodNotAllowed,
				expectedBody:   "Method not allowed\n",
			},
			{
				name:           "GET /other returns not found",
				path:           "/other",
				method:         http.MethodGet,
				expectedStatus: http.StatusNotFound,
				expectedBody:   "Not Found",
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				req, err := http.NewRequest(tt.method, tt.path, nil)
				if err != nil {
					t.Fatalf("failed to create request: %v", err)
				}

				rr := &testResponseWriter{
					statusCode: 200,
					body:       make([]byte, 0),
					headers:    make(http.Header),
				}

				handler.ServeHTTP(rr, req)

				if rr.statusCode != tt.expectedStatus {
					t.Errorf("expected status %d, got %d", tt.expectedStatus, rr.statusCode)
				}

				if string(rr.body) != tt.expectedBody {
					t.Errorf("expected body %q, got %q", tt.expectedBody, string(rr.body))
				}
			})
		}
	})
}

// testResponseWriter is a simple implementation of http.ResponseWriter for testing
type testResponseWriter struct {
	statusCode int
	body       []byte
	headers    http.Header
}

func (w *testResponseWriter) Header() http.Header {
	return w.headers
}

func (w *testResponseWriter) Write(data []byte) (int, error) {
	w.body = append(w.body, data...)
	return len(data), nil
}

func (w *testResponseWriter) WriteHeader(statusCode int) {
	w.statusCode = statusCode
}

func TestInternalAddrIncrement(t *testing.T) {
	// Test that internal addresses are properly incremented
	testCases := []struct {
		input    string
		expected string
	}{
		{"localhost:8000", "localhost:8001"},
		{"127.0.0.1:9999", "127.0.0.1:10000"},
		{"example.com:80", "example.com:81"},
	}

	for _, tc := range testCases {
		t.Run(fmt.Sprintf("increment_%s", tc.input), func(t *testing.T) {
			result, err := getInternalAddr(tc.input)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if result != tc.expected {
				t.Errorf("expected %s, got %s", tc.expected, result)
			}
		})
	}
}

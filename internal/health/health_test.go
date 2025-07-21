package health

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestHandler(t *testing.T) {
	config := Config{
		ServiceName: "test-service",
		Version:     "1.0.0",
	}

	handler := Handler(config)

	tests := []struct {
		name           string
		method         string
		expectedStatus int
		expectedBody   bool // whether to check response body
	}{
		{
			name:           "GET request returns health status",
			method:         http.MethodGet,
			expectedStatus: http.StatusOK,
			expectedBody:   true,
		},
		{
			name:           "POST request returns method not allowed",
			method:         http.MethodPost,
			expectedStatus: http.StatusMethodNotAllowed,
			expectedBody:   false,
		},
		{
			name:           "PUT request returns method not allowed",
			method:         http.MethodPut,
			expectedStatus: http.StatusMethodNotAllowed,
			expectedBody:   false,
		},
		{
			name:           "DELETE request returns method not allowed",
			method:         http.MethodDelete,
			expectedStatus: http.StatusMethodNotAllowed,
			expectedBody:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, "/healthz", nil)
			w := httptest.NewRecorder()

			handler(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, w.Code)
			}

			if tt.expectedBody {
				// Check Content-Type header
				contentType := w.Header().Get("Content-Type")
				if contentType != "application/json" {
					t.Errorf("expected Content-Type application/json, got %s", contentType)
				}

				// Parse and validate response body
				var response Response
				if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
					t.Errorf("failed to unmarshal response: %v", err)
				}

				if response.Status != StatusHealthy {
					t.Errorf("expected status %s, got %s", StatusHealthy, response.Status)
				}

				if response.Service != config.ServiceName {
					t.Errorf("expected service %s, got %s", config.ServiceName, response.Service)
				}

				if response.Version != config.Version {
					t.Errorf("expected version %s, got %s", config.Version, response.Version)
				}

				// Check that timestamp is recent (within last 5 seconds)
				now := time.Now().UTC()
				if response.Timestamp.After(now) || response.Timestamp.Before(now.Add(-5*time.Second)) {
					t.Errorf("timestamp %v is not recent", response.Timestamp)
				}
			}
		})
	}
}

func TestSimpleHandler(t *testing.T) {
	handler := SimpleHandler()

	tests := []struct {
		name           string
		method         string
		expectedStatus int
		expectedBody   string
	}{
		{
			name:           "GET request returns OK",
			method:         http.MethodGet,
			expectedStatus: http.StatusOK,
			expectedBody:   "OK",
		},
		{
			name:           "POST request returns method not allowed",
			method:         http.MethodPost,
			expectedStatus: http.StatusMethodNotAllowed,
			expectedBody:   "Method not allowed\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, "/health/liveness", nil)
			w := httptest.NewRecorder()

			handler(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, w.Code)
			}

			body := w.Body.String()
			if body != tt.expectedBody {
				t.Errorf("expected body %q, got %q", tt.expectedBody, body)
			}
		})
	}
}

func TestResponseStructure(t *testing.T) {
	config := Config{
		ServiceName: "mcp-grafana",
		Version:     "v1.2.3",
	}

	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	w := httptest.NewRecorder()

	handler := Handler(config)
	handler(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}

	var response Response
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	// Test all fields are populated correctly
	if response.Status != StatusHealthy {
		t.Errorf("expected status %s, got %s", StatusHealthy, response.Status)
	}

	if response.Service != config.ServiceName {
		t.Errorf("expected service %s, got %s", config.ServiceName, response.Service)
	}

	if response.Version != config.Version {
		t.Errorf("expected version %s, got %s", config.Version, response.Version)
	}

	if response.Timestamp.IsZero() {
		t.Error("timestamp should not be zero")
	}

	// Ensure the response can be marshaled back to JSON
	_, err := json.Marshal(response)
	if err != nil {
		t.Errorf("response should be JSON serializable: %v", err)
	}
}

func TestStatus(t *testing.T) {
	tests := []struct {
		name   string
		status Status
		str    string
	}{
		{
			name:   "healthy status",
			status: StatusHealthy,
			str:    "healthy",
		},
		{
			name:   "unhealthy status",
			status: StatusUnhealthy,
			str:    "unhealthy",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if string(tt.status) != tt.str {
				t.Errorf("expected %s, got %s", tt.str, string(tt.status))
			}
		})
	}
}

func BenchmarkHandler(b *testing.B) {
	config := Config{
		ServiceName: "bench-test",
		Version:     "1.0.0",
	}
	handler := Handler(config)

	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		handler(w, req)
	}
}

func BenchmarkSimpleHandler(b *testing.B) {
	handler := SimpleHandler()
	req := httptest.NewRequest(http.MethodGet, "/health/liveness", nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		handler(w, req)
	}
}

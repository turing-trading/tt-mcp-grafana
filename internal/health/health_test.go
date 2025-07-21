package health

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHandler(t *testing.T) {
	handler := Handler()

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
		{
			name:           "PUT request returns method not allowed",
			method:         http.MethodPut,
			expectedStatus: http.StatusMethodNotAllowed,
			expectedBody:   "Method not allowed\n",
		},
		{
			name:           "DELETE request returns method not allowed",
			method:         http.MethodDelete,
			expectedStatus: http.StatusMethodNotAllowed,
			expectedBody:   "Method not allowed\n",
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

			body := w.Body.String()
			if body != tt.expectedBody {
				t.Errorf("expected body %q, got %q", tt.expectedBody, body)
			}
		})
	}
}

func TestHealthCheckEndpoint(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	w := httptest.NewRecorder()

	handler := Handler()
	handler(w, req)

	// Verify response
	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	expectedBody := "OK"
	if body := w.Body.String(); body != expectedBody {
		t.Errorf("expected body %q, got %q", expectedBody, body)
	}
}

func BenchmarkHandler(b *testing.B) {
	handler := Handler()
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		handler(w, req)
	}
}

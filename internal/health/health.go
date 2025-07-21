package health

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"time"
)

// Status represents the health status of the service
type Status string

const (
	StatusHealthy   Status = "healthy"
	StatusUnhealthy Status = "unhealthy"
)

// Response represents the health check response
type Response struct {
	Status    Status    `json:"status"`
	Service   string    `json:"service"`
	Version   string    `json:"version"`
	Timestamp time.Time `json:"timestamp"`
}

// Config holds the configuration for health checks
type Config struct {
	ServiceName string
	Version     string
}

// Handler creates an HTTP handler for health checks
func Handler(config Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		response := Response{
			Status:    StatusHealthy,
			Service:   config.ServiceName,
			Version:   config.Version,
			Timestamp: time.Now().UTC(),
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		if err := json.NewEncoder(w).Encode(response); err != nil {
			slog.Error("Failed to encode health check response", "error", err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}

		slog.Debug("Health check requested", "remote_addr", r.RemoteAddr, "user_agent", r.UserAgent())
	}
}

// SimpleHandler creates a minimal health check handler that just returns 200 OK
func SimpleHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	}
}

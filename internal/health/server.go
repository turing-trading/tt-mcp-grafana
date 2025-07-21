package health

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"sync"
	"time"
)

// Server represents a health check server
type Server struct {
	config     Config
	httpServer *http.Server
	mux        *http.ServeMux
	mu         sync.RWMutex
	started    bool
}

// NewServer creates a new health check server
func NewServer(config Config) *Server {
	mux := http.NewServeMux()

	// Add health check endpoints
	mux.HandleFunc("/healthz", Handler(config))
	mux.HandleFunc("/health", Handler(config))
	mux.HandleFunc("/health/readiness", Handler(config))
	mux.HandleFunc("/health/liveness", SimpleHandler())

	return &Server{
		config: config,
		mux:    mux,
	}
}

// Start starts the health check server on the specified address
func (s *Server) Start(addr string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.started {
		return fmt.Errorf("health server already started")
	}

	s.httpServer = &http.Server{
		Addr:         addr,
		Handler:      s.mux,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	s.started = true
	slog.Info("Starting health check server", "address", addr)

	if err := s.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		s.started = false
		return fmt.Errorf("health server failed to start: %w", err)
	}

	return nil
}

// StartAsync starts the health check server asynchronously
func (s *Server) StartAsync(addr string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.started {
		return fmt.Errorf("health server already started")
	}

	s.httpServer = &http.Server{
		Addr:         addr,
		Handler:      s.mux,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		s.started = true
		slog.Info("Starting health check server", "address", addr)

		if err := s.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("Health server failed", "error", err)
			s.mu.Lock()
			s.started = false
			s.mu.Unlock()
		}
	}()

	return nil
}

// Stop gracefully stops the health check server
func (s *Server) Stop(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.started || s.httpServer == nil {
		return nil
	}

	slog.Info("Stopping health check server")
	err := s.httpServer.Shutdown(ctx)
	s.started = false
	return err
}

// IsStarted returns whether the health server is currently running
func (s *Server) IsStarted() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.started
}

// GetHealthPort extracts a separate port for health checks based on the main server port
func GetHealthPort(mainAddr string) (string, error) {
	host, port, err := net.SplitHostPort(mainAddr)
	if err != nil {
		return "", fmt.Errorf("invalid address format: %w", err)
	}

	// Parse the port number
	var mainPort int
	if _, err := fmt.Sscanf(port, "%d", &mainPort); err != nil {
		return "", fmt.Errorf("invalid port number: %w", err)
	}

	// Use main port + 1000 for health checks to avoid conflicts
	healthPort := mainPort + 1000
	return fmt.Sprintf("%s:%d", host, healthPort), nil
}

// GetAvailablePort finds an available port on the system
func GetAvailablePort() (int, error) {
	listener, err := net.Listen("tcp", ":0")
	if err != nil {
		return 0, err
	}
	defer listener.Close()

	addr := listener.Addr().(*net.TCPAddr)
	return addr.Port, nil
}

// GenerateHealthAddr generates a health check address based on the main address
func GenerateHealthAddr(mainAddr string) string {
	// Try to get a predictable health port first
	if healthAddr, err := GetHealthPort(mainAddr); err == nil {
		return healthAddr
	}

	// Fallback to finding any available port
	host, _, err := net.SplitHostPort(mainAddr)
	if err != nil {
		host = "localhost"
	}

	if port, err := GetAvailablePort(); err == nil {
		return fmt.Sprintf("%s:%d", host, port)
	}

	// Last resort fallback
	return "localhost:9001"
}

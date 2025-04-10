package tools

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestMCPListAlloyComponents(t *testing.T) {
	// Mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v0/web/components" {
			t.Errorf("Expected to request '/api/v0/web/components', got: %s", r.URL.Path)
		}

		components := []AlloyComponent{
			{
				Name:    "loki.write",
				Type:    "block",
				LocalID: "loki.write.default",
				Label:   "default",
				Health: Health{
					State:   "healthy",
					Message: "started component",
				},
			},
		}

		json.NewEncoder(w).Encode(components)
	}))
	defer server.Close()

	// Test the function
	ctx := context.Background()
	result, err := MCPListAlloyComponents(ctx)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	// Basic validation of the output format
	if result == "" {
		t.Error("Expected non-empty result")
	}
}

func TestMCPGetAlloyComponentDetails(t *testing.T) {
	// Mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v0/web/components/loki.write.default" {
			t.Errorf("Expected to request '/api/v0/web/components/loki.write.default', got: %s", r.URL.Path)
		}

		component := AlloyComponent{
			Name:    "loki.write",
			Type:    "block",
			LocalID: "loki.write.default",
			Label:   "default",
			Health: Health{
				State:       "healthy",
				Message:     "started component",
				UpdatedTime: time.Now(),
			},
			ReferencesTo: []string{"loki.source.docker.default"},
		}

		json.NewEncoder(w).Encode(component)
	}))
	defer server.Close()

	// Test the function
	ctx := context.Background()
	result, err := MCPGetAlloyComponentDetails(ctx, "loki.write.default")
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	// Basic validation of the output format
	if result == "" {
		t.Error("Expected non-empty result")
	}
}

func TestMCPAnalyzeAlloyPipeline(t *testing.T) {
	// Mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v0/web/components" {
			t.Errorf("Expected to request '/api/v0/web/components', got: %s", r.URL.Path)
		}

		components := []AlloyComponent{
			{
				Name:    "loki.write",
				Type:    "block",
				LocalID: "loki.write.default",
				Label:   "default",
				Health: Health{
					State:   "healthy",
					Message: "started component",
				},
			},
			{
				Name:         "loki.source.docker",
				Type:         "block",
				LocalID:      "loki.source.docker.default",
				Label:        "default",
				ReferencesTo: []string{"loki.write.default"},
				Health: Health{
					State:   "healthy",
					Message: "started component",
				},
			},
		}

		json.NewEncoder(w).Encode(components)
	}))
	defer server.Close()

	// Test the function
	ctx := context.Background()
	result, err := MCPAnalyzeAlloyPipeline(ctx, "loki")
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	// Basic validation of the output format
	if result == "" {
		t.Error("Expected non-empty result")
	}
}

func TestMCPGetAlloyComponentHealth(t *testing.T) {
	// Mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v0/web/components" {
			t.Errorf("Expected to request '/api/v0/web/components', got: %s", r.URL.Path)
		}

		components := []AlloyComponent{
			{
				Name:    "loki.write",
				Type:    "block",
				LocalID: "loki.write.default",
				Label:   "default",
				Health: Health{
					State:       "healthy",
					Message:     "started component",
					UpdatedTime: time.Now(),
				},
			},
		}

		json.NewEncoder(w).Encode(components)
	}))
	defer server.Close()

	// Test the function
	ctx := context.Background()
	result, err := MCPGetAlloyComponentHealth(ctx)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	// Basic validation of the output format
	if result == "" {
		t.Error("Expected non-empty result")
	}
}

package tools

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetAlloyVersion(t *testing.T) {
	// Mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/metrics" {
			t.Errorf("Expected to request '/metrics', got: %s", r.URL.Path)
		}

		// Return mock metrics response
		w.Write([]byte(`# TYPE alloy_build_info gauge
alloy_build_info{branch="HEAD",goarch="arm64",goos="darwin",goversion="go1.23.5",revision="unknown",tags="builtinassets,noebpf",version="v1.6.1"} 1`))
	}))
	defer server.Close()

	t.Setenv(alloyHostEnvVar, server.URL[7:]) // Remove "http://" prefix

	// Test the function
	ctx := context.Background()
	version, err := GetAlloyVersion(ctx)
	require.NoError(t, err)
	assert.Equal(t, "1", version.Major)
	assert.Equal(t, "6", version.Minor)
}

func TestMCPGetAlloyComponentDocs(t *testing.T) {
	// Mock servers for both metrics and component details
	metricsServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/metrics" {
			t.Errorf("Expected to request '/metrics', got: %s", r.URL.Path)
		}
		w.Write([]byte(`# TYPE alloy_build_info gauge
alloy_build_info{version="v1.6.1"} 1`))
	}))
	defer metricsServer.Close()

	componentsServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v0/web/components/loki.process.default" {
			t.Errorf("Expected to request '/api/v0/web/components/loki.process.default', got: %s", r.URL.Path)
		}
		w.Write([]byte(`{
            "name": "loki.process",
            "type": "block",
            "localID": "loki.process.default",
            "label": "default"
        }`))
	}))
	defer componentsServer.Close()

	t.Setenv(alloyHostEnvVar, metricsServer.URL[7:]) // Remove "http://" prefix

	// Test the function
	ctx := context.Background()
	req := AlloyDocsRequest{
		ComponentID: "loki.process.default",
		Section:     "stagestructured_metadata",
	}
	result, err := MCPGetAlloyComponentDocs(ctx, req)

	require.NoError(t, err)
	assert.Contains(t, result, "https://grafana.com/docs/alloy/v1.6/reference/components/loki/loki.process#stagestructured_metadata")
}

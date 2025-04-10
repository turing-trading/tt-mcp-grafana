package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	mcpgrafana "github.com/grafana/mcp-grafana"
	"github.com/mark3labs/mcp-go/server"
)

const (
	defaultAlloyHost = "localhost:12345"
	alloyHostEnvVar  = "ALLOY_HOST"
)

// AlloyComponent represents a component in the Alloy configuration
type AlloyComponent struct {
	Name                 string   `json:"name"`
	Type                 string   `json:"type"`
	LocalID              string   `json:"localID"`
	ModuleID             string   `json:"moduleID"`
	Label                string   `json:"label"`
	ReferencesTo         []string `json:"referencesTo"`
	ReferencedBy         []string `json:"referencedBy"`
	Health               Health   `json:"health"`
	Original             string   `json:"original"`
	Arguments            []string `json:"arguments"`
	Exports              []string `json:"exports"`
	DebugInfo            []string `json:"debugInfo"`
	LiveDebuggingEnabled bool     `json:"liveDebuggingEnabled"`
}

// Health represents the health status of an Alloy component
type Health struct {
	State       string    `json:"state"`
	Message     string    `json:"message"`
	UpdatedTime time.Time `json:"updatedTime"`
}

// AlloyComponentRequest represents a request for component details
type AlloyComponentRequest struct {
	ComponentID string `json:"component_id" jsonschema:"required,description=The ID of the component to get details for"`
}

// AlloyPipelineRequest represents a request for pipeline analysis
type AlloyPipelineRequest struct {
	PipelineType string `json:"pipeline_type" jsonschema:"required,description=The type of pipeline to analyze (loki, prometheus, otel)"`
}

// EmptyParams represents an empty parameter struct for tools that don't need input
type EmptyParams struct{}

// ListAlloyComponents is a tool for listing all components in Alloy
var ListAlloyComponents = mcpgrafana.MustTool[EmptyParams, string](
	"mcp_alloy_list_components",
	"List all components in the Alloy instance",
	func(ctx context.Context, _ EmptyParams) (string, error) {
		return MCPListAlloyComponents(ctx)
	},
)

// GetAlloyComponentDetails is a tool for getting component details
var GetAlloyComponentDetails = mcpgrafana.MustTool[AlloyComponentRequest, string](
	"mcp_alloy_get_component_details",
	"Get detailed information about a specific component",
	func(ctx context.Context, req AlloyComponentRequest) (string, error) {
		return MCPGetAlloyComponentDetails(ctx, req.ComponentID)
	},
)

// AnalyzeAlloyPipeline is a tool for analyzing pipeline components
var AnalyzeAlloyPipeline = mcpgrafana.MustTool[AlloyPipelineRequest, string](
	"mcp_alloy_analyze_pipeline",
	"Analyze components of a specific type (loki, prometheus, otel)",
	func(ctx context.Context, req AlloyPipelineRequest) (string, error) {
		return MCPAnalyzeAlloyPipeline(ctx, req.PipelineType)
	},
)

// GetAlloyComponentHealth is a tool for getting component health
var GetAlloyComponentHealth = mcpgrafana.MustTool[EmptyParams, string](
	"mcp_alloy_get_health",
	"Get health status of all components",
	func(ctx context.Context, _ EmptyParams) (string, error) {
		return MCPGetAlloyComponentHealth(ctx)
	},
)

// AddAlloyTools registers all Alloy tools with the MCP server
func AddAlloyTools(mcp *server.MCPServer) {
	ListAlloyComponents.Register(mcp)
	GetAlloyComponentDetails.Register(mcp)
	AnalyzeAlloyPipeline.Register(mcp)
	GetAlloyComponentHealth.Register(mcp)
}

// ListAlloyComponents lists all components from a running Alloy instance
func ListAlloyComponentsFunc(ctx context.Context) ([]AlloyComponent, error) {
	host := os.Getenv(alloyHostEnvVar)
	if host == "" {
		host = defaultAlloyHost
	}
	baseURL := fmt.Sprintf("http://%s", host)
	url := fmt.Sprintf("%s/api/v0/web/components", baseURL)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		// Check specifically for connection refused error
		if strings.Contains(err.Error(), "connection refused") {
			return nil, fmt.Errorf("connection refused: Alloy service appears to be down or not running on %s. Please ensure the service is running and try again", baseURL)
		}
		return nil, fmt.Errorf("fetching components: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var components []AlloyComponent
	if err := json.NewDecoder(resp.Body).Decode(&components); err != nil {
		return nil, fmt.Errorf("decoding response: %w", err)
	}

	return components, nil
}

// GetAlloyComponentDetails gets detailed information about a specific component
func GetAlloyComponentDetailsFunc(ctx context.Context, componentID string) (*AlloyComponent, error) {
	host := os.Getenv(alloyHostEnvVar)
	if host == "" {
		host = defaultAlloyHost
	}
	baseURL := fmt.Sprintf("http://%s", host)
	url := fmt.Sprintf("%s/api/v0/web/components/%s", baseURL, componentID)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetching component details: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var component AlloyComponent
	if err := json.NewDecoder(resp.Body).Decode(&component); err != nil {
		return nil, fmt.Errorf("decoding response: %w", err)
	}

	return &component, nil
}

// AnalyzeAlloyPipeline analyzes components of a specific type and their connections
func AnalyzeAlloyPipelineFunc(ctx context.Context, pipelineType string) (map[string]interface{}, error) {
	components, err := ListAlloyComponentsFunc(ctx)
	if err != nil {
		return nil, fmt.Errorf("listing components: %w", err)
	}

	var pipelineComponents []AlloyComponent
	for _, component := range components {
		if isComponentOfType(component, pipelineType) {
			pipelineComponents = append(pipelineComponents, component)
		}
	}

	// Build a map of connections and metadata
	analysis := map[string]interface{}{
		"components":  pipelineComponents,
		"connections": buildConnectionMap(pipelineComponents),
		"health":      analyzeHealth(pipelineComponents),
	}

	return analysis, nil
}

// GetAlloyComponentHealth gets the health status of components
func GetAlloyComponentHealthFunc(ctx context.Context) (map[string]Health, error) {
	components, err := ListAlloyComponentsFunc(ctx)
	if err != nil {
		return nil, fmt.Errorf("listing components: %w", err)
	}

	healthMap := make(map[string]Health)
	for _, component := range components {
		healthMap[component.LocalID] = component.Health
	}

	return healthMap, nil
}

// Helper functions

func isComponentOfType(component AlloyComponent, pipelineType string) bool {
	switch pipelineType {
	case "loki":
		return component.Name == "loki.write" ||
			component.Name == "loki.source.docker" ||
			component.Name == "loki.source.file"
	case "prometheus":
		return component.Name == "prometheus.remote_write" ||
			component.Name == "prometheus.scrape"
	case "otel":
		return component.Name == "otelcol.receiver.otlp" ||
			component.Name == "otelcol.processor.batch" ||
			component.Name == "otelcol.exporter.otlp"
	default:
		return false
	}
}

func buildConnectionMap(components []AlloyComponent) map[string][]string {
	connections := make(map[string][]string)
	for _, component := range components {
		connections[component.LocalID] = append(
			connections[component.LocalID],
			component.ReferencesTo...,
		)
	}
	return connections
}

func analyzeHealth(components []AlloyComponent) map[string]string {
	health := make(map[string]string)
	for _, component := range components {
		health[component.LocalID] = component.Health.State
	}
	return health
}

// MCP Tool Functions

func MCPListAlloyComponents(ctx context.Context) (string, error) {
	components, err := ListAlloyComponentsFunc(ctx)
	if err != nil {
		return "", err
	}

	// Format the output in a way that's useful for an LLM
	var result string
	result = "Alloy Components:\n\n"

	for _, c := range components {
		result += fmt.Sprintf("Component: %s (ID: %s)\n", c.Name, c.LocalID)
		result += fmt.Sprintf("  Type: %s\n", c.Type)
		result += fmt.Sprintf("  Label: %s\n", c.Label)
		result += fmt.Sprintf("  Health: %s (%s)\n", c.Health.State, c.Health.Message)
		if len(c.ReferencesTo) > 0 {
			result += fmt.Sprintf("  References: %v\n", c.ReferencesTo)
		}
		if len(c.ReferencedBy) > 0 {
			result += fmt.Sprintf("  Referenced By: %v\n", c.ReferencedBy)
		}
		result += "\n"
	}

	return result, nil
}

func MCPGetAlloyComponentDetails(ctx context.Context, componentID string) (string, error) {
	component, err := GetAlloyComponentDetailsFunc(ctx, componentID)
	if err != nil {
		return "", err
	}

	// Format the detailed output
	result := fmt.Sprintf("Details for Component %s:\n\n", componentID)
	result += fmt.Sprintf("Name: %s\n", component.Name)
	result += fmt.Sprintf("Type: %s\n", component.Type)
	result += fmt.Sprintf("Label: %s\n", component.Label)
	result += fmt.Sprintf("Module ID: %s\n", component.ModuleID)
	result += fmt.Sprintf("Health State: %s\n", component.Health.State)
	result += fmt.Sprintf("Health Message: %s\n", component.Health.Message)
	result += fmt.Sprintf("Last Updated: %s\n", component.Health.UpdatedTime)

	if len(component.ReferencesTo) > 0 {
		result += "\nReferences To:\n"
		for _, ref := range component.ReferencesTo {
			result += fmt.Sprintf("- %s\n", ref)
		}
	}

	if len(component.ReferencedBy) > 0 {
		result += "\nReferenced By:\n"
		for _, ref := range component.ReferencedBy {
			result += fmt.Sprintf("- %s\n", ref)
		}
	}

	if len(component.Arguments) > 0 {
		result += "\nArguments:\n"
		for _, arg := range component.Arguments {
			result += fmt.Sprintf("- %s\n", arg)
		}
	}

	if len(component.Exports) > 0 {
		result += "\nExports:\n"
		for _, exp := range component.Exports {
			result += fmt.Sprintf("- %s\n", exp)
		}
	}

	return result, nil
}

func MCPAnalyzeAlloyPipeline(ctx context.Context, pipelineType string) (string, error) {
	analysis, err := AnalyzeAlloyPipelineFunc(ctx, pipelineType)
	if err != nil {
		return "", err
	}

	// Format the analysis in a way that's useful for understanding the pipeline
	result := fmt.Sprintf("Analysis of %s Pipeline:\n\n", pipelineType)

	// Add components
	if components, ok := analysis["components"].([]AlloyComponent); ok {
		result += "Components:\n"
		for _, c := range components {
			result += fmt.Sprintf("- %s (ID: %s)\n", c.Name, c.LocalID)
			result += fmt.Sprintf("  Health: %s\n", c.Health.State)
		}
		result += "\n"
	}

	// Add connections
	if connections, ok := analysis["connections"].(map[string][]string); ok {
		result += "Connections:\n"
		for id, refs := range connections {
			result += fmt.Sprintf("- %s connects to: %v\n", id, refs)
		}
		result += "\n"
	}

	// Add health summary
	if health, ok := analysis["health"].(map[string]string); ok {
		result += "Health Summary:\n"
		for id, state := range health {
			result += fmt.Sprintf("- %s: %s\n", id, state)
		}
	}

	return result, nil
}

func MCPGetAlloyComponentHealth(ctx context.Context) (string, error) {
	health, err := GetAlloyComponentHealthFunc(ctx)
	if err != nil {
		return "", err
	}

	// Format the health information
	result := "Component Health Status:\n\n"

	for id, h := range health {
		result += fmt.Sprintf("Component: %s\n", id)
		result += fmt.Sprintf("  State: %s\n", h.State)
		result += fmt.Sprintf("  Message: %s\n", h.Message)
		result += fmt.Sprintf("  Last Updated: %s\n", h.UpdatedTime)
		result += "\n"
	}

	return result, nil
}

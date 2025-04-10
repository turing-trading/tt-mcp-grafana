package tools

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"regexp"
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
	Name                 string      `json:"name"`
	Type                 string      `json:"type"`
	LocalID              string      `json:"localID"`
	ModuleID             string      `json:"moduleID"`
	Label                string      `json:"label"`
	ReferencesTo         []string    `json:"referencesTo"`
	ReferencedBy         []string    `json:"referencedBy"`
	Health               Health      `json:"health"`
	Original             string      `json:"original"`
	Arguments            interface{} `json:"arguments"`
	Exports              interface{} `json:"exports"`
	DebugInfo            []string    `json:"debugInfo"`
	LiveDebuggingEnabled bool        `json:"liveDebuggingEnabled"`
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

// AlloyVersion represents version information from metrics
type AlloyVersion struct {
	Major   string
	Minor   string
	Version string
}

// AlloyDocsRequest represents a request for component documentation
type AlloyDocsRequest struct {
	ComponentID string `json:"component_id" jsonschema:"required,description=The ID of the component to get documentation for"`
	Section     string `json:"section,omitempty" jsonschema:"description=Optional section anchor in the documentation"`
}

// ListAlloyComponents is a tool for listing all components in Alloy
var ListAlloyComponents = mcpgrafana.MustTool[EmptyParams, string](
	"alloy_list_components",
	"List all components in the Alloy instance",
	func(ctx context.Context, _ EmptyParams) (string, error) {
		return MCPListAlloyComponents(ctx)
	},
)

// GetAlloyComponentDetails is a tool for getting component details
var GetAlloyComponentDetails = mcpgrafana.MustTool[AlloyComponentRequest, string](
	"alloy_get_component_details",
	"Get detailed information about a specific component",
	func(ctx context.Context, req AlloyComponentRequest) (string, error) {
		slog.DebugContext(ctx, "Entering GetAlloyComponentDetails tool handler lambda", "component_id", req.ComponentID)
		result, err := MCPGetAlloyComponentDetails(ctx, req.ComponentID)
		if err != nil {
			slog.ErrorContext(ctx, "Error executing MCPGetAlloyComponentDetails within lambda", "error", err, "component_id", req.ComponentID)
		} else {
			slog.DebugContext(ctx, "Exiting GetAlloyComponentDetails tool handler lambda successfully", "component_id", req.ComponentID)
		}
		return result, err
	},
)

// AnalyzeAlloyPipeline is a tool for analyzing pipeline components
var AnalyzeAlloyPipeline = mcpgrafana.MustTool[AlloyPipelineRequest, string](
	"alloy_analyze_pipeline",
	"Analyze components of a specific type (loki, prometheus, otel)",
	func(ctx context.Context, req AlloyPipelineRequest) (string, error) {
		return MCPAnalyzeAlloyPipeline(ctx, req.PipelineType)
	},
)

// GetAlloyComponentHealth is a tool for getting component health
var GetAlloyComponentHealth = mcpgrafana.MustTool[EmptyParams, string](
	"alloy_get_health",
	"Get health status of all components",
	func(ctx context.Context, _ EmptyParams) (string, error) {
		return MCPGetAlloyComponentHealth(ctx)
	},
)

// GetAlloyVersion fetches and parses the version from metrics endpoint
func GetAlloyVersion(ctx context.Context) (*AlloyVersion, error) {
	host := os.Getenv(alloyHostEnvVar)
	if host == "" {
		host = defaultAlloyHost
	}
	baseURL := fmt.Sprintf("http://%s", host)
	url := fmt.Sprintf("%s/metrics", baseURL)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetching metrics: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.Contains(line, "alloy_build_info") {
			// Parse version from the metrics line
			// Regex captures major, minor, and patch version numbers
			matches := regexp.MustCompile(`version="v(\d+)\.(\d+)\.(\d+)"`).FindStringSubmatch(line)
			if len(matches) == 4 { // Expect 4 matches: full string, major, minor, patch
				return &AlloyVersion{
					Major:   matches[1],
					Minor:   matches[2],
					Version: fmt.Sprintf("v%s.%s.%s", matches[1], matches[2], matches[3]), // Construct full version string
				}, nil
			}
		}
	}

	return nil, fmt.Errorf("version information not found in metrics")
}

// MCPGetAlloyComponentDocs fetches the raw markdown documentation for a component from GitHub.
func MCPGetAlloyComponentDocs(ctx context.Context, req AlloyDocsRequest) (string, error) {
	slog.DebugContext(ctx, "MCPGetAlloyComponentDocs called", "component_id", req.ComponentID)
	// Get Alloy version
	version, err := GetAlloyVersion(ctx)
	if err != nil {
		slog.ErrorContext(ctx, "Failed to get Alloy version for docs", "error", err)
		return "", fmt.Errorf("getting Alloy version: %w", err)
	}
	slog.DebugContext(ctx, "Got Alloy version for docs", "version", version.Version)

	// Get component details primarily to get the canonical component name (e.g., discovery.relabel)
	component, err := GetAlloyComponentDetailsFunc(ctx, req.ComponentID)
	if err != nil {
		// If getting details fails, maybe the component ID itself is the name (e.g., user typed 'discovery.docker')
		slog.WarnContext(ctx, "Failed to get component details for docs, attempting to use component ID as name", "component_id", req.ComponentID, "error", err)
		component = &AlloyComponent{Name: req.ComponentID} // Use the requested ID as the name
		// Don't return error here, proceed with the potentially incorrect name
	}

	// Split component name to get type and name (e.g., "discovery.relabel" -> ["discovery", "relabel"])
	parts := strings.SplitN(component.Name, ".", 2)
	if len(parts) != 2 {
		slog.ErrorContext(ctx, "Invalid component name format for docs URL", "component_name", component.Name)
		return "", fmt.Errorf("invalid component name format: %s", component.Name)
	}
	componentType := parts[0]
	componentFileName := component.Name // Use the full name like discovery.relabel

	// Construct the URL to the raw markdown file on GitHub for the specific release branch
	// Example: https://raw.githubusercontent.com/grafana/alloy/release/v1.6/docs/sources/reference/components/discovery/discovery.relabel.md
	githubURL := fmt.Sprintf("https://raw.githubusercontent.com/grafana/alloy/release/v%s.%s/docs/sources/reference/components/%s/%s.md",
		version.Major, version.Minor, componentType, componentFileName)

	slog.DebugContext(ctx, "Constructed GitHub raw content URL", "url", githubURL)

	// Fetch the markdown content
	httpReq, err := http.NewRequestWithContext(ctx, "GET", githubURL, nil)
	if err != nil {
		slog.ErrorContext(ctx, "Failed to create HTTP request for GitHub docs", "error", err, "url", githubURL)
		return "", fmt.Errorf("creating request for GitHub docs: %w", err)
	}

	httpResp, err := http.DefaultClient.Do(httpReq)
	if err != nil {
		slog.ErrorContext(ctx, "Failed to execute HTTP request for GitHub docs", "error", err, "url", githubURL)
		return "", fmt.Errorf("fetching GitHub docs: %w", err)
	}
	defer httpResp.Body.Close()

	slog.DebugContext(ctx, "Received response for GitHub docs", "url", githubURL, "status_code", httpResp.StatusCode)

	if httpResp.StatusCode != http.StatusOK {
		emsg := fmt.Sprintf("failed to fetch docs from GitHub, received status %d for %s", httpResp.StatusCode, githubURL)
		slog.ErrorContext(ctx, "Received non-OK status code for GitHub docs", "status_code", httpResp.StatusCode, "url", githubURL)
		// Try fetching from 'main' branch as a fallback
		githubURLMain := fmt.Sprintf("https://raw.githubusercontent.com/grafana/alloy/main/docs/sources/reference/components/%s/%s.md",
			componentType, componentFileName)
		slog.WarnContext(ctx, "Retrying GitHub docs fetch from main branch", "url", githubURLMain)
		httpReqMain, _ := http.NewRequestWithContext(ctx, "GET", githubURLMain, nil)
		httpRespMain, errMain := http.DefaultClient.Do(httpReqMain)
		if errMain != nil || httpRespMain.StatusCode != http.StatusOK {
			closeBody(httpRespMain) // Close body if resp exists
			slog.ErrorContext(ctx, "Failed to fetch docs from GitHub main branch as well", "error", errMain, "status_code", getStatusCode(httpRespMain))
			return "", fmt.Errorf(emsg) // Return original error
		}
		defer httpRespMain.Body.Close()
		httpResp = httpRespMain // Use the response from main branch
		slog.InfoContext(ctx, "Successfully fetched docs from GitHub main branch after release branch failed")
	}

	// Read the response body
	markdownBytes, err := io.ReadAll(httpResp.Body)
	if err != nil {
		slog.ErrorContext(ctx, "Failed to read GitHub docs response body", "error", err, "url", githubURL)
		return "", fmt.Errorf("reading GitHub docs response: %w", err)
	}

	markdownContent := string(markdownBytes)
	slog.DebugContext(ctx, "Successfully fetched and read markdown content", "component", component.Name, "chars", len(markdownContent))

	// Optional: Could add some basic formatting or indicate source?
	result := fmt.Sprintf("## Documentation for %s (Alloy %s)\n\nSource: %s\n\n---\n\n%s",
		component.Name, version.Version, githubURL, markdownContent)

	return result, nil
}

// Helper to safely close body
func closeBody(resp *http.Response) {
	if resp != nil && resp.Body != nil {
		resp.Body.Close()
	}
}

// Helper to safely get status code
func getStatusCode(resp *http.Response) int {
	if resp != nil {
		return resp.StatusCode
	}
	return 0
}

// GetAlloyComponentDocs is a tool for getting component documentation
var GetAlloyComponentDocs = mcpgrafana.MustTool[AlloyDocsRequest, string](
	"alloy_get_component_docs",
	"Get documentation for a specific component based on the running Alloy version",
	MCPGetAlloyComponentDocs,
)

// AddAlloyTools registers all Alloy tools with the MCP server
func AddAlloyTools(mcp *server.MCPServer) {
	ListAlloyComponents.Register(mcp)
	GetAlloyComponentDetails.Register(mcp)
	AnalyzeAlloyPipeline.Register(mcp)
	GetAlloyComponentHealth.Register(mcp)
	GetAlloyComponentDocs.Register(mcp)
}

// GetAlloyComponentDetails gets detailed information about a specific component
func GetAlloyComponentDetailsFunc(ctx context.Context, componentID string) (*AlloyComponent, error) {
	host := os.Getenv(alloyHostEnvVar)
	if host == "" {
		host = defaultAlloyHost
	}
	baseURL := fmt.Sprintf("http://%s", host)
	url := fmt.Sprintf("%s/api/v0/web/components/%s", baseURL, componentID)

	slog.DebugContext(ctx, "Attempting to fetch component details", "url", url)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		slog.ErrorContext(ctx, "Failed to create HTTP request for component details", "error", err, "url", url)
		return nil, fmt.Errorf("creating request: %w", err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		// Check specifically for connection refused error
		if strings.Contains(err.Error(), "connection refused") {
			return nil, fmt.Errorf("connection refused: Alloy service appears to be down or not running on %s. Please ensure the service is running and try again", baseURL)
		}
		slog.ErrorContext(ctx, "Failed to execute HTTP request for component details", "error", err, "url", url)
		return nil, fmt.Errorf("fetching component details: %w", err)
	}
	defer resp.Body.Close()

	slog.DebugContext(ctx, "Received response for component details", "url", url, "status_code", resp.StatusCode)

	if resp.StatusCode != http.StatusOK {
		slog.ErrorContext(ctx, "Received non-OK status code for component details", "url", url, "status_code", resp.StatusCode)
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var component AlloyComponent
	if err := json.NewDecoder(resp.Body).Decode(&component); err != nil {
		slog.ErrorContext(ctx, "Failed to decode component details response", "error", err, "url", url)
		return nil, fmt.Errorf("decoding response: %w", err)
	}

	slog.DebugContext(ctx, "Successfully decoded component details", "component_id", componentID, "decoded_component", fmt.Sprintf("%+v", component))

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
	// Split the component name by dot and check if the first part matches the pipeline type
	parts := strings.Split(component.Name, ".")
	if len(parts) < 2 {
		return false
	}
	return parts[0] == pipelineType
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
	slog.DebugContext(ctx, "MCPGetAlloyComponentDetails called", "component_id", componentID)
	component, err := GetAlloyComponentDetailsFunc(ctx, componentID)
	if err != nil {
		slog.ErrorContext(ctx, "Error getting component details from Func", "error", err, "component_id", componentID)
		return "", err
	}

	slog.DebugContext(ctx, "Formatting component details", "component_id", componentID)

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

	if component.Arguments != nil {
		result += "\nArguments:\n"
		result += fmt.Sprintf("- %+v\n", component.Arguments)
	}

	if component.Exports != nil {
		result += "\nExports:\n"
		result += fmt.Sprintf("- %+v\n", component.Exports)
	}

	if len(component.DebugInfo) > 0 {
		result += "\nDebug Info:\n"
		for _, dbg := range component.DebugInfo {
			result += fmt.Sprintf("- %s\n", dbg)
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

// ListAlloyComponentsFunc lists all components from a running Alloy instance
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

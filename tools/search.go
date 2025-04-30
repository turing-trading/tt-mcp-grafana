package tools

import (
	"context"
	"fmt"

	"github.com/mark3labs/mcp-go/server"

	"github.com/grafana/grafana-openapi-client-go/client/search"
	"github.com/grafana/grafana-openapi-client-go/models"
	mcpgrafana "github.com/grafana/mcp-grafana"
)

var dashboardTypeStr = "dash-db"

type SearchDashboardsParams struct {
	Query     string   `json:"query,omitempty" jsonschema:"description=Single keyword or phrase for case-insensitive substring matching within the dashboard title. The search looks for the *exact* substring provided. Do NOT combine multiple terms."`
	Tag       []string `json:"tag,omitempty" jsonschema:"description=List of dashboard tags to search for. Must be an array\\, even if searching for a single tag. Example: [\"production\"]"`
	UID       []string `json:"dashboardUID,omitempty" jsonschema:"description=List of dashboard UIDs to search for. Must be an array\\, even if searching for a single UID. Example: [\"abc-123\"]"`
	FolderUID []string `json:"folderUID,omitempty" jsonschema:"description=List of folder UIDs to search in. Must be an array\\, even if searching for a single UID. Example: [\"folder-xyz\"]"`
}

func searchDashboards(ctx context.Context, args SearchDashboardsParams) (models.HitList, error) {
	c := mcpgrafana.GrafanaClientFromContext(ctx)
	params := search.NewSearchParamsWithContext(ctx)
	params.SetType(&dashboardTypeStr)

	if args.Query != "" {
		params.SetQuery(&args.Query)
	}
	if len(args.Tag) > 0 {
		params.SetTag(args.Tag)
	}
	if len(args.UID) > 0 {
		params.SetDashboardUIDs(args.UID)
	}
	if len(args.FolderUID) > 0 {
		params.SetFolderUIDs(args.FolderUID)
	}

	searchResult, err := c.Search.Search(params)
	if err != nil {
		return nil, fmt.Errorf("failed to search dashboards with params %+v: %w", args, err)
	}
	return searchResult.Payload, nil
}

var SearchDashboards = mcpgrafana.MustTool(
	"search_dashboards",
	"Search for Grafana dashboards by title query, tags, UIDs, or folder UID. Returns a list of matching dashboards with details like title, UID, folder, tags, and URL.",
	searchDashboards,
)

func AddSearchTools(mcp *server.MCPServer) {
	SearchDashboards.Register(mcp)
}

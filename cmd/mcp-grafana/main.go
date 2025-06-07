package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"slices"
	"strings"

	"github.com/mark3labs/mcp-go/server"

	mcpgrafana "github.com/grafana/mcp-grafana"
	"github.com/grafana/mcp-grafana/tools"
)

func maybeAddTools(s *server.MCPServer, tf func(*server.MCPServer, mcpgrafana.ToolMode), enabledTools []string, disable bool, category string, mode mcpgrafana.ToolMode) {
	if !slices.Contains(enabledTools, category) {
		slog.Debug("Not enabling tools", "category", category)
		return
	}
	if disable {
		slog.Info("Disabling tools", "category", category)
		return
	}
	slog.Debug("Enabling tools", "category", category)
	tf(s, mode)
}

// disabledTools indicates whether each category of tools should be disabled.
type disabledTools struct {
	enabledTools string

	search, datasource, incident,
	prometheus, loki, alerting,
	dashboard, oncall, asserts, sift, admin bool
}

// Configuration for the Grafana client.
type grafanaConfig struct {
	// Whether to enable debug mode for the Grafana transport.
	debug bool
}

func (dt *disabledTools) addFlags() {
	flag.StringVar(&dt.enabledTools, "enabled-tools", "search,datasource,incident,prometheus,loki,alerting,dashboard,oncall,asserts,sift,admin", "A comma separated list of tools enabled for this server. Can be overwritten entirely or by disabling specific components, e.g. --disable-search.")

	flag.BoolVar(&dt.search, "disable-search", false, "Disable search tools")
	flag.BoolVar(&dt.datasource, "disable-datasource", false, "Disable datasource tools")
	flag.BoolVar(&dt.incident, "disable-incident", false, "Disable incident tools")
	flag.BoolVar(&dt.prometheus, "disable-prometheus", false, "Disable prometheus tools")
	flag.BoolVar(&dt.loki, "disable-loki", false, "Disable loki tools")
	flag.BoolVar(&dt.alerting, "disable-alerting", false, "Disable alerting tools")
	flag.BoolVar(&dt.dashboard, "disable-dashboard", false, "Disable dashboard tools")
	flag.BoolVar(&dt.oncall, "disable-oncall", false, "Disable oncall tools")
	flag.BoolVar(&dt.asserts, "disable-asserts", false, "Disable asserts tools")
	flag.BoolVar(&dt.sift, "disable-sift", false, "Disable sift tools")
	flag.BoolVar(&dt.admin, "disable-admin", false, "Disable admin tools")
}

func (gc *grafanaConfig) addFlags() {
	flag.BoolVar(&gc.debug, "debug", false, "Enable debug mode for the Grafana transport")
}

func (dt *disabledTools) addTools(s *server.MCPServer, mode mcpgrafana.ToolMode) {
	enabledTools := strings.Split(dt.enabledTools, ",")
	maybeAddTools(s, tools.AddSearchTools, enabledTools, dt.search, "search", mode)
	maybeAddTools(s, tools.AddDatasourceTools, enabledTools, dt.datasource, "datasource", mode)
	maybeAddTools(s, tools.AddIncidentTools, enabledTools, dt.incident, "incident", mode)
	maybeAddTools(s, tools.AddPrometheusTools, enabledTools, dt.prometheus, "prometheus", mode)
	maybeAddTools(s, tools.AddLokiTools, enabledTools, dt.loki, "loki", mode)
	maybeAddTools(s, tools.AddAlertingTools, enabledTools, dt.alerting, "alerting", mode)
	maybeAddTools(s, tools.AddDashboardTools, enabledTools, dt.dashboard, "dashboard", mode)
	maybeAddTools(s, tools.AddOnCallTools, enabledTools, dt.oncall, "oncall", mode)
	maybeAddTools(s, tools.AddAssertsTools, enabledTools, dt.asserts, "asserts", mode)
	maybeAddTools(s, tools.AddSiftTools, enabledTools, dt.sift, "sift", mode)
	maybeAddTools(s, tools.AddAdminTools, enabledTools, dt.admin, "admin", mode)
}

func newServer(dt disabledTools, mode mcpgrafana.ToolMode) *server.MCPServer {
	s := server.NewMCPServer(
		"mcp-grafana",
		"0.1.0",
	)
	dt.addTools(s, mode)
	return s
}

func run(transport, addr, basePath string, logLevel slog.Level, dt disabledTools, gc grafanaConfig, mode mcpgrafana.ToolMode) error {
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: logLevel})))
	s := newServer(dt, mode)

	switch transport {
	case "stdio":
		srv := server.NewStdioServer(s)
		srv.SetContextFunc(mcpgrafana.ComposedStdioContextFunc(gc.debug))
		slog.Info("Starting Grafana MCP server using stdio transport")
		return srv.Listen(context.Background(), os.Stdin, os.Stdout)
	case "sse":
		srv := server.NewSSEServer(s,
			server.WithSSEContextFunc(mcpgrafana.ComposedSSEContextFunc(gc.debug)),
			server.WithStaticBasePath(basePath),
		)
		slog.Info("Starting Grafana MCP server using SSE transport", "address", addr, "basePath", basePath)
		if err := srv.Start(addr); err != nil {
			return fmt.Errorf("Server error: %v", err)
		}
	default:
		return fmt.Errorf(
			"Invalid transport type: %s. Must be 'stdio' or 'sse'",
			transport,
		)
	}
	return nil
}

func main() {
	var transport string
	flag.StringVar(&transport, "t", "stdio", "Transport type (stdio or sse)")
	flag.StringVar(
		&transport,
		"transport",
		"stdio",
		"Transport type (stdio or sse)",
	)
	addr := flag.String("sse-address", "localhost:8000", "The host and port to start the sse server on")
	basePath := flag.String("base-path", "", "Base path for the sse server")
	logLevel := flag.String("log-level", "info", "Log level (debug, info, warn, error)")
	var modeStr string
	flag.StringVar(&modeStr, "mode", "write", "Tool mode (read or write)")
	var dt disabledTools
	dt.addFlags()
	var gc grafanaConfig
	gc.addFlags()
	flag.Parse()

	mode := mcpgrafana.ToolMode(modeStr)

	if err := run(transport, *addr, *basePath, parseLevel(*logLevel), dt, gc, mode); err != nil {
		panic(err)
	}
}

func parseLevel(level string) slog.Level {
	var l slog.Level
	if err := l.UnmarshalText([]byte(level)); err != nil {
		return slog.LevelInfo
	}
	return l
}

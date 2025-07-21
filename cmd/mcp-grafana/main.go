package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"runtime/debug"
	"slices"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/mark3labs/mcp-go/server"

	mcpgrafana "github.com/grafana/mcp-grafana"
	"github.com/grafana/mcp-grafana/internal/health"
	"github.com/grafana/mcp-grafana/tools"
)

// version returns the version of the mcp-grafana binary.
// It is populated by the `runtime/debug` package which
// fetches git information from the build directory.
var version = sync.OnceValue(func() string {
	// Default version string returned by `runtime/debug` if built
	// from the source repository rather than with `go install`.
	v := "(devel)"
	if bi, ok := debug.ReadBuildInfo(); ok {
		v = bi.Main.Version
	}
	return v
})

func maybeAddTools(s *server.MCPServer, tf func(*server.MCPServer), enabledTools []string, disable bool, category string) {
	if !slices.Contains(enabledTools, category) {
		slog.Debug("Not enabling tools", "category", category)
		return
	}
	if disable {
		slog.Info("Disabling tools", "category", category)
		return
	}
	slog.Debug("Enabling tools", "category", category)
	tf(s)
}

// disabledTools indicates whether each category of tools should be disabled.
type disabledTools struct {
	enabledTools string

	search, datasource, incident,
	prometheus, loki, alerting,
	dashboard, oncall, asserts, sift, admin,
	pyroscope bool
}

// Configuration for the Grafana client.
type grafanaConfig struct {
	// Whether to enable debug mode for the Grafana transport.
	debug bool

	// TLS configuration
	tlsCertFile   string
	tlsKeyFile    string
	tlsCAFile     string
	tlsSkipVerify bool
}

// Configuration for health checks.
type healthConfig struct {
	enabled      bool
	port         string
	separatePort bool
}

func (dt *disabledTools) addFlags() {
	flag.StringVar(&dt.enabledTools, "enabled-tools", "search,datasource,incident,prometheus,loki,alerting,dashboard,oncall,asserts,sift,admin,pyroscope", "A comma separated list of tools enabled for this server. Can be overwritten entirely or by disabling specific components, e.g. --disable-search.")

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
	flag.BoolVar(&dt.pyroscope, "disable-pyroscope", false, "Disable pyroscope tools")
}

func (gc *grafanaConfig) addFlags() {
	flag.BoolVar(&gc.debug, "debug", false, "Enable debug mode for the Grafana transport")

	// TLS configuration flags
	flag.StringVar(&gc.tlsCertFile, "tls-cert-file", "", "Path to TLS certificate file for client authentication")
	flag.StringVar(&gc.tlsKeyFile, "tls-key-file", "", "Path to TLS private key file for client authentication")
	flag.StringVar(&gc.tlsCAFile, "tls-ca-file", "", "Path to TLS CA certificate file for server verification")
	flag.BoolVar(&gc.tlsSkipVerify, "tls-skip-verify", false, "Skip TLS certificate verification (insecure)")
}

func (hc *healthConfig) addFlags() {
	flag.BoolVar(&hc.enabled, "health-enabled", true, "Enable health check endpoints for server transports")
	flag.StringVar(&hc.port, "health-port", "", "Port for health check endpoints (defaults to main port + 1000)")
	flag.BoolVar(&hc.separatePort, "health-separate-port", true, "Run health checks on a separate port")
}

func (dt *disabledTools) addTools(s *server.MCPServer) {
	enabledTools := strings.Split(dt.enabledTools, ",")
	maybeAddTools(s, tools.AddSearchTools, enabledTools, dt.search, "search")
	maybeAddTools(s, tools.AddDatasourceTools, enabledTools, dt.datasource, "datasource")
	maybeAddTools(s, tools.AddIncidentTools, enabledTools, dt.incident, "incident")
	maybeAddTools(s, tools.AddPrometheusTools, enabledTools, dt.prometheus, "prometheus")
	maybeAddTools(s, tools.AddLokiTools, enabledTools, dt.loki, "loki")
	maybeAddTools(s, tools.AddAlertingTools, enabledTools, dt.alerting, "alerting")
	maybeAddTools(s, tools.AddDashboardTools, enabledTools, dt.dashboard, "dashboard")
	maybeAddTools(s, tools.AddOnCallTools, enabledTools, dt.oncall, "oncall")
	maybeAddTools(s, tools.AddAssertsTools, enabledTools, dt.asserts, "asserts")
	maybeAddTools(s, tools.AddSiftTools, enabledTools, dt.sift, "sift")
	maybeAddTools(s, tools.AddAdminTools, enabledTools, dt.admin, "admin")
	maybeAddTools(s, tools.AddPyroscopeTools, enabledTools, dt.pyroscope, "pyroscope")
}

func newServer(dt disabledTools) *server.MCPServer {
	s := server.NewMCPServer("mcp-grafana", version(), server.WithInstructions(`
	This server provides access to your Grafana instance and the surrounding ecosystem.

	Available Capabilities:
	- Dashboards: Search, retrieve, update, and create dashboards. Extract panel queries and datasource information.
	- Datasources: List and fetch details for datasources.
	- Prometheus & Loki: Run PromQL and LogQL queries, retrieve metric/log metadata, and explore label names/values.
	- Incidents: Search, create, update, and resolve incidents in Grafana Incident.
	- Sift Investigations: Start and manage Sift investigations, analyze logs/traces, find error patterns, and detect slow requests.
	- Alerting: List and fetch alert rules and notification contact points.
	- OnCall: View and manage on-call schedules, shifts, teams, and users.
	- Admin: List teams and perform administrative tasks.
	- Pyroscope: Profile applications and fetch profiling data.
	`))
	dt.addTools(s)
	return s
}

func run(transport, addr, basePath, endpointPath string, logLevel slog.Level, dt disabledTools, gc mcpgrafana.GrafanaConfig, hc healthConfig) error {
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: logLevel})))
	s := newServer(dt)

	// Set up signal handling for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	var healthServer *health.Server

	switch transport {
	case "stdio":
		srv := server.NewStdioServer(s)
		srv.SetContextFunc(mcpgrafana.ComposedStdioContextFunc(gc))
		slog.Info("Starting Grafana MCP server using stdio transport", "version", version())
		return srv.Listen(context.Background(), os.Stdin, os.Stdout)
	case "sse":
		srv := server.NewSSEServer(s,
			server.WithSSEContextFunc(mcpgrafana.ComposedSSEContextFunc(gc)),
			server.WithStaticBasePath(basePath),
		)

		// Start health check server if enabled
		if hc.enabled {
			healthConfig := health.Config{
				ServiceName: "mcp-grafana",
				Version:     version(),
			}
			healthServer = health.NewServer(healthConfig)

			healthAddr := addr
			if hc.separatePort {
				if hc.port != "" {
					healthAddr = hc.port
				} else {
					healthAddr = health.GenerateHealthAddr(addr)
				}
			}

			if err := healthServer.StartAsync(healthAddr); err != nil {
				slog.Error("Failed to start health server", "error", err)
			} else {
				slog.Info("Health check endpoints available", "address", healthAddr, "endpoints", "/healthz, /health, /health/readiness, /health/liveness")
			}
		}

		slog.Info("Starting Grafana MCP server using SSE transport", "version", version(), "address", addr, "basePath", basePath)
		go func() {
			if err := srv.Start(addr); err != nil {
				slog.Error("SSE server error", "error", err)
				cancel()
			}
		}()
	case "streamable-http":
		srv := server.NewStreamableHTTPServer(s, server.WithHTTPContextFunc(mcpgrafana.ComposedHTTPContextFunc(gc)),
			server.WithStateLess(true),
			server.WithEndpointPath(endpointPath),
		)

		// Start health check server if enabled
		if hc.enabled {
			healthConfig := health.Config{
				ServiceName: "mcp-grafana",
				Version:     version(),
			}
			healthServer = health.NewServer(healthConfig)

			healthAddr := addr
			if hc.separatePort {
				if hc.port != "" {
					healthAddr = hc.port
				} else {
					healthAddr = health.GenerateHealthAddr(addr)
				}
			}

			if err := healthServer.StartAsync(healthAddr); err != nil {
				slog.Error("Failed to start health server", "error", err)
			} else {
				slog.Info("Health check endpoints available", "address", healthAddr, "endpoints", "/healthz, /health, /health/readiness, /health/liveness")
			}
		}

		slog.Info("Starting Grafana MCP server using StreamableHTTP transport", "version", version(), "address", addr, "endpointPath", endpointPath)
		go func() {
			if err := srv.Start(addr); err != nil {
				slog.Error("StreamableHTTP server error", "error", err)
				cancel()
			}
		}()
	default:
		return fmt.Errorf(
			"Invalid transport type: %s. Must be 'stdio', 'sse' or 'streamable-http'",
			transport,
		)
	}

	// Wait for shutdown signal for non-stdio transports
	if transport != "stdio" {
		select {
		case <-sigChan:
			slog.Info("Received shutdown signal")
		case <-ctx.Done():
			slog.Info("Context cancelled")
		}

		// Graceful shutdown
		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer shutdownCancel()

		if healthServer != nil {
			if err := healthServer.Stop(shutdownCtx); err != nil {
				slog.Error("Error stopping health server", "error", err)
			}
		}
	}

	return nil
}

func main() {
	var transport string
	flag.StringVar(&transport, "t", "stdio", "Transport type (stdio, sse or streamable-http)")
	flag.StringVar(
		&transport,
		"transport",
		"stdio",
		"Transport type (stdio, sse or streamable-http)",
	)
	addr := flag.String("address", "localhost:8000", "The host and port to start the sse server on")
	basePath := flag.String("base-path", "", "Base path for the sse server")
	endpointPath := flag.String("endpoint-path", "/mcp", "Endpoint path for the streamable-http server")
	logLevel := flag.String("log-level", "info", "Log level (debug, info, warn, error)")
	showVersion := flag.Bool("version", false, "Print the version and exit")
	var dt disabledTools
	dt.addFlags()
	var gc grafanaConfig
	gc.addFlags()
	var hc healthConfig
	hc.addFlags()
	flag.Parse()

	if *showVersion {
		fmt.Println(version())
		os.Exit(0)
	}

	// Convert local grafanaConfig to mcpgrafana.GrafanaConfig
	grafanaConfig := mcpgrafana.GrafanaConfig{Debug: gc.debug}
	if gc.tlsCertFile != "" || gc.tlsKeyFile != "" || gc.tlsCAFile != "" || gc.tlsSkipVerify {
		grafanaConfig.TLSConfig = &mcpgrafana.TLSConfig{
			CertFile:   gc.tlsCertFile,
			KeyFile:    gc.tlsKeyFile,
			CAFile:     gc.tlsCAFile,
			SkipVerify: gc.tlsSkipVerify,
		}
	}

	if err := run(transport, *addr, *basePath, *endpointPath, parseLevel(*logLevel), dt, grafanaConfig, hc); err != nil {
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

package mcptools

import (
	"io"
	"log/slog"

	"github.com/redhat-appstudio/helmet/api"
	"github.com/redhat-appstudio/helmet/internal/chartfs"
	"github.com/redhat-appstudio/helmet/internal/flags"
	"github.com/redhat-appstudio/helmet/internal/integrations"
	"github.com/redhat-appstudio/helmet/internal/k8s"
)

// MCPToolsContext holds the dependencies needed to create MCP tools.
// This context is populated by the framework and passed to the builder function.
// The logger is automatically configured to write to io.Discard because MCP
// server uses STDIO for protocol communication. Any output to stdout/stderr will
// corrupt the MCP protocol messages.
type MCPToolsContext struct {
	AppCtx             *api.AppContext       // application context
	Logger             *slog.Logger          // discard logger
	Flags              *flags.Flags          // global flags
	ChartFS            *chartfs.ChartFS      // embedded filesystem
	Kube               *k8s.Kube             // kubernetes client
	IntegrationManager *integrations.Manager // integrations manager
	Image              string                // installer's container image
}

// NewMCPToolsContext creates a new MCPToolsContext with a logger configured for
// MCP server operation.
func NewMCPToolsContext(
	appCtx *api.AppContext,
	f *flags.Flags,
	cfs *chartfs.ChartFS,
	kube *k8s.Kube,
	integrationManager *integrations.Manager,
	image string,
) MCPToolsContext {
	return MCPToolsContext{
		AppCtx: appCtx,
		// CRITICAL: Logger MUST use io.Discard for MCP STDIO protocol compatibility
		Logger:             f.GetLogger(io.Discard),
		Flags:              f,
		ChartFS:            cfs,
		Kube:               kube,
		IntegrationManager: integrationManager,
		Image:              image,
	}
}

// MCPToolsBuilder is a function that creates MCP tools given a context.
// Consumers can provide custom builders to customize which tools are registered.
type MCPToolsBuilder func(MCPToolsContext) ([]Interface, error)

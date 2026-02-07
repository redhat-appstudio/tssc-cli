package mcptools

import (
	"io"

	"github.com/redhat-appstudio/helmet/api"
	"github.com/redhat-appstudio/helmet/internal/flags"
	"github.com/redhat-appstudio/helmet/internal/integrations"
	"github.com/redhat-appstudio/helmet/internal/runcontext"
)

// MCPToolsContext holds the dependencies needed to create MCP tools.
// This context is populated by the framework and passed to the builder function.
// The logger is automatically configured to write to io.Discard because MCP
// server uses STDIO for protocol communication. Any output to stdout/stderr will
// corrupt the MCP protocol messages.
//
//nolint:revive
type MCPToolsContext struct {
	*runcontext.RunContext
	AppContext         *api.AppContext       // application identity
	Flags              *flags.Flags          // global flags
	IntegrationManager *integrations.Manager // integrations manager
	Image              string                // installer's container image
}

// NewMCPToolsContext creates a new MCPToolsContext with a logger configured for
// MCP server operation.
func NewMCPToolsContext(
	appCtx *api.AppContext,
	runCtx *runcontext.RunContext,
	f *flags.Flags,
	integrationManager *integrations.Manager,
	image string,
) MCPToolsContext {
	mcpRunCtx := &runcontext.RunContext{
		Kube:    runCtx.Kube,
		ChartFS: runCtx.ChartFS,
		// CRITICAL: Logger MUST use io.Discard for MCP STDIO protocol compatibility
		Logger: f.GetLogger(io.Discard),
	}
	return MCPToolsContext{
		RunContext:         mcpRunCtx,
		AppContext:         appCtx,
		Flags:              f,
		IntegrationManager: integrationManager,
		Image:              image,
	}
}

// MCPToolsBuilder is a function that creates MCP tools given a context.
// Consumers can provide custom builders to customize which tools are registered.
//
//nolint:revive
type MCPToolsBuilder func(MCPToolsContext) ([]Interface, error)

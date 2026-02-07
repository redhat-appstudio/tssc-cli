package subcmd

import (
	"github.com/redhat-appstudio/helmet/internal/config"
	"github.com/redhat-appstudio/helmet/internal/installer"
	"github.com/redhat-appstudio/helmet/internal/mcptools"
	"github.com/redhat-appstudio/helmet/internal/resolver"
)

// StandardMCPToolsBuilder returns a builder function that creates all standard
// MCP tools for the installer api. These tools provide the core MCP
// functionality.
func StandardMCPToolsBuilder() mcptools.MCPToolsBuilder {
	return standardMCPTools
}

// standardMCPTools is the actual builder implementation.
func standardMCPTools(
	toolsCtx mcptools.MCPToolsContext,
) ([]mcptools.Interface, error) {
	cm := config.NewConfigMapManager(toolsCtx.Kube, toolsCtx.AppContext.Name)

	// Config tools.
	configTools, err := mcptools.NewConfigTools(
		toolsCtx.AppContext,
		toolsCtx.Logger,
		toolsCtx.ChartFS,
		toolsCtx.Kube,
		cm,
	)
	if err != nil {
		return nil, err
	}

	// Topology builder (shared dependency).
	tb, err := resolver.NewTopologyBuilder(
		toolsCtx.AppContext,
		toolsCtx.Logger,
		toolsCtx.ChartFS,
		toolsCtx.IntegrationManager,
	)
	if err != nil {
		return nil, err
	}

	// Job manager (shared dependency).
	job := installer.NewJob(toolsCtx.AppContext, toolsCtx.Kube)

	// Status tool.
	statusTool := mcptools.NewStatusTool(toolsCtx.AppContext.Name, cm, tb, job)

	// Integration tools, creates its own instance for metadata introspection.
	integrationCmd := NewIntegration(
		toolsCtx.AppContext,
		toolsCtx.RunContext,
		toolsCtx.IntegrationManager,
	)
	integrationTools := mcptools.NewIntegrationTools(
		toolsCtx.AppContext.Name, integrationCmd, cm, toolsCtx.IntegrationManager,
	)

	// Deploy tools.
	deployTools := mcptools.NewDeployTools(
		toolsCtx.AppContext.Name, cm, tb, job, toolsCtx.Image)

	// Notes tool.
	notesTool := mcptools.NewNotesTool(
		toolsCtx.AppContext.Name,
		toolsCtx.Logger,
		toolsCtx.Flags,
		toolsCtx.Kube,
		cm,
		tb,
		job,
	)

	// Topology tool
	topologyTool := mcptools.NewTopologyTool(
		toolsCtx.AppContext.Name, toolsCtx.ChartFS, cm, tb)

	return []mcptools.Interface{
		configTools,
		statusTool,
		integrationTools,
		deployTools,
		notesTool,
		topologyTool,
	}, nil
}

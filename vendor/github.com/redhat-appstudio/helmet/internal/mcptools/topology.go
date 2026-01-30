package mcptools

import (
	"bytes"
	"context"
	"fmt"

	"github.com/redhat-appstudio/helmet/internal/chartfs"
	"github.com/redhat-appstudio/helmet/internal/config"
	"github.com/redhat-appstudio/helmet/internal/resolver"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// TopologyTool represents the MCP tool that's responsible to report
// the dependency topology of the installer based on the cluster
// configuration and Helm charts.
type TopologyTool struct {
	appName string                    // application name
	cfs     *chartfs.ChartFS          // embedded filesystem
	cm      *config.ConfigMapManager  // cluster configuration
	tb      *resolver.TopologyBuilder // topology builder
}

const (
	// topologySuffix mcp topology tool name suffix
	topologySuffix = "_topology"
)

// topologyHandler shows a table of the topology.
func (t *TopologyTool) topologyHandler(
	ctx context.Context,
	_ mcp.CallToolRequest,
) (*mcp.CallToolResult, error) {
	// Load the installer configuration from the cluster.
	cfg, err := t.cm.GetConfig(ctx)
	if err != nil {
		return nil, err
	}
	// Inspect the topology to ensure all dependencies and
	// integrations are resolved.
	if _, err := t.tb.Build(ctx, cfg); err != nil {
		return nil, err
	}
	// Resolving the dependency topology based on the installer configuration and
	// Helm charts.
	r := resolver.NewResolver(cfg, t.tb.GetCollection(), resolver.NewTopology())
	if err := r.Resolve(); err != nil {
		return nil, err
	}

	var buf bytes.Buffer
	r.Print(&buf)

	return mcp.NewToolResultText(fmt.Sprintf(`
The topology is a table with following columns:

  - Index: the index of the chart in the dependency graph.
  - Dependency: the name of the Helm chart.
  - Namespace: the OpenShift namespace where the chart is installed.
  - Product: the name of the product the chart is associated with.
  - Depends-On: comma-separated list of charts the chart depends on.
  - Provided-Integrations: comma-separated integrations provided by the chart.
  - Required-Integrations: CEL expressions with the required integrations.

---
%s`,
		buf.String())), nil
}

func (t *TopologyTool) Init(s *server.MCPServer) {
	s.AddTools([]server.ServerTool{{
		Tool: mcp.NewTool(
			t.appName+topologySuffix,
			mcp.WithDescription(`
Report the dependency topology of the installer based on the
cluster configuration and installer dependencies (Helm charts).
			`),
		),
		Handler: t.topologyHandler,
	}}...)
}

// NewTopologyTool instantiates a new TopologyTool.
func NewTopologyTool(
	appName string,
	cfs *chartfs.ChartFS,
	cm *config.ConfigMapManager,
	tb *resolver.TopologyBuilder,
) *TopologyTool {
	return &TopologyTool{
		appName: appName,
		cfs:     cfs,
		cm:      cm,
		tb:      tb,
	}
}

package mcptools

import (
	"context"
	"fmt"

	"github.com/redhat-appstudio/tssc-cli/pkg/config"
	"github.com/redhat-appstudio/tssc-cli/pkg/constants"
	"github.com/redhat-appstudio/tssc-cli/pkg/installer"
	"github.com/redhat-appstudio/tssc-cli/pkg/resolver"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// DeployTools represents the tools used for deploying the TSSC using the
// installer on a container image, and running in the cluster, using a Kubernetes
// Job.
type DeployTools struct {
	cm              *config.ConfigMapManager  // cluster configuration
	topologyBuilder *resolver.TopologyBuilder // topology builder
	job             *installer.Job            // cluster deployment job
	image           string                    // tssc container image
}

var _ Interface = &DeployTools{}

// DeployToolName deploy tool name.
const DeployToolName = constants.AppName + "_deploy"

// deployHandler handles the deployment of TSSC components.
func (d *DeployTools) deployHandler(
	ctx context.Context,
	ctr mcp.CallToolRequest,
) (*mcp.CallToolResult, error) {
	// Ensure the cluster is configured, if the ConfigMap is not found, creates a
	// error to inform the user about MCP configuration tools.
	cfg, err := d.cm.GetConfig(ctx)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf(`
The cluster is not configured yet, use the tool %q to identify the cluster
installation status, and the next actions after that.

> %s`,
			StatusToolName, err.Error(),
		)), nil
	}

	if err = d.job.Create(ctx, cfg.Installer.Namespace, d.image); err != nil {
		return nil, fmt.Errorf("failed to create installer job: %w", err)
	}

	// Command to get the logs of the deployment job.
	logsCmd := d.job.GetJobLogFollowCmd(cfg.Installer.Namespace)
	return mcp.NewToolResultText(fmt.Sprintf(`
The installer job has been created successfully. Use the tool 'tssc_deploy_status'
to check the deployment status using the MCP server.

You can follow the logs by running:

	%s`,
		logsCmd,
	)), nil
}

// Init registers the deployment tools on the MCP server.
func (d *DeployTools) Init(mcpServer *server.MCPServer) {
	mcpServer.AddTools([]server.ServerTool{{
		Tool: mcp.NewTool(
			DeployToolName,
			mcp.WithDescription(`
Deploys TSSC components to the cluster, uses the cluster configuration to deploy
the TSSC components sequentially.`,
			),
		),
		Handler: d.deployHandler,
	}}...)
}

// NewDeployTools creates a new DeployTools instance.
func NewDeployTools(
	cm *config.ConfigMapManager,
	job *installer.Job,
	image string,
) *DeployTools {
	return &DeployTools{cm: cm, job: job, image: image}
}

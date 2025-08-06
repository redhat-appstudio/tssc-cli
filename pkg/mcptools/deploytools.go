package mcptools

import (
	"context"
	"fmt"
	"strings"

	"github.com/redhat-appstudio/tssc-cli/pkg/config"
	"github.com/redhat-appstudio/tssc-cli/pkg/installer"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// DeployTools represents the tools used for deploying the TSSC using the
// installer on a container image, and running in the cluster, using a Kubernetes
// Job.
type DeployTools struct {
	cm    *config.ConfigMapManager // cluster configuration
	job   *installer.Job           // cluster deployment job
	image string                   // tssc container image
}

// statusHandler handles the status of the deployment job. It checks if the
// cluster deployment job is running or completed.
func (d *DeployTools) statusHandler(
	ctx context.Context,
	ctr mcp.CallToolRequest,
) (*mcp.CallToolResult, error) {
	// Ensure the cluster is configured, if the ConfigMap is not found, creates a
	// message to inform the user about MCP configuration tools.
	cfg, err := d.cm.GetConfig(ctx)
	if err != nil {
		return mcp.NewToolResultText(fmt.Sprintf(`
The cluster is not configured yet , use the tool 'tssc_config_create' to configure
it. That's the first step to deploy TSSC components.

Inspecting the configuration in the cluster returned the following error:

%s`,
			err.Error(),
		)), nil
	}

	// Given the cluster is configured, inspect the current state of the
	// deployment job.
	state, err := d.job.GetState(ctx)
	if err != nil {
		return nil, err
	}

	// Command to get the logs of the deployment job.
	logsCmd := d.job.GetJobLogFollowCmd(cfg.Installer.Namespace)

	// Handle different states of the deployment job.
	switch state {
	case installer.NotFound:
		return mcp.NewToolResultText(`
The cluster is ready to deploy the TSSC components. Use the tool "tssc_deploy" to
deploy the components.`,
		), nil
	case installer.Deploying:
		return mcp.NewToolResultText(fmt.Sprintf(`
The cluster is deploying the TSSC components. Please wait for the deployment to
complete. You can use the following command to follow the deployment job logs:

	%s`,
			logsCmd,
		)), nil
	case installer.Failed:
		return mcp.NewToolResultError(fmt.Sprintf(`
The deployment job has failed. You can use the following command to view the
related POD logs:

	%s`,
			logsCmd,
		)), nil
	case installer.Done:
		return mcp.NewToolResultText(fmt.Sprintf(`
The TSSC components have been deployed successfully. You can use the following
command to inspect the installation logs and get initial information for each
product deployed:

	%s`,
			logsCmd,
		)), nil
	}
	return nil, fmt.Errorf("unknown deployment state %q", state)
}

// deployHandler handles the deployment of TSSC components.
func (d *DeployTools) deployHandler(
	ctx context.Context,
	ctr mcp.CallToolRequest,
) (*mcp.CallToolResult, error) {
	// Ensure the cluster is configured, if the ConfigMap is not found, creates a
	// error to inform the user about MCP configuration tools.
	cfg, err := d.cm.GetConfig(ctx)
	if err != nil {
		errMsg := strings.Builder{}
		errMsg.WriteString("The cluster is not configured yet , use the tool")
		errMsg.WriteString(" 'tssc_config_create' to configure it")
		return nil, fmt.Errorf("%s: %s", errMsg.String(), err)
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
func (d *DeployTools) Init(s *server.MCPServer) {
	s.AddTools([]server.ServerTool{{
		// TODO: the installer status will be moved to a dedicatd function,
		// "tssc_status", see RHTAP-4826 for more details. While this MCP function
		// only shows the deploy job status, the future "tssc_status" will include
		// the installed Helm charts and more.
		Tool: mcp.NewTool(
			"tssc_deploy_status",
			mcp.WithDescription(`
Reports the status of the TSSC deploy Job running in the cluster.`),
		),
		Handler: d.statusHandler,
	}, {
		Tool: mcp.NewTool(
			"tssc_deploy",
			mcp.WithDescription(`
Deploys TSSC components to the cluster, uses the cluster configuration to deploy
the TSSC components sequentially.`,
			),
		),
		Handler: d.deployHandler,
	}}...)
}

// NewDeployTools creates a new DeployTools instance.l
func NewDeployTools(
	cm *config.ConfigMapManager, // cluster configuration manager
	job *installer.Job, // job manager instance
	image string, // container image
) *DeployTools {
	return &DeployTools{cm: cm, job: job, image: image}
}

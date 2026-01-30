package mcptools

import (
	"context"
	"fmt"

	"github.com/redhat-appstudio/helmet/internal/config"
	"github.com/redhat-appstudio/helmet/internal/installer"
	"github.com/redhat-appstudio/helmet/internal/resolver"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// DeployTools represents the tools used for deploying the components using the
// installer on a container image, and running in the cluster, using a Kubernetes
// Job.
type DeployTools struct {
	appName         string                    // application name
	cm              *config.ConfigMapManager  // cluster configuration
	topologyBuilder *resolver.TopologyBuilder // topology builder
	job             *installer.Job            // cluster deployment job
	image           string                    // installer container image
}

var _ Interface = &DeployTools{}

const (
	// deploySuffix deploy tool name suffix.
	deploySuffix = "_deploy"

	// DebugArg enables debug mode for the deployment job.
	DebugArg = "debug"
	// DryRunArg runs the deployment job on dry-run.
	DryRunArg = "dry-run"
	// ForceArg forces the recreation of the deployment job.
	ForceArg = "force"
)

// deployHandler handles the deployment of components.
func (d *DeployTools) deployHandler(
	ctx context.Context,
	ctr mcp.CallToolRequest,
) (*mcp.CallToolResult, error) {
	// Ensure the cluster is configured, if theConfigMap is not found, creates a
	// error to inform the user about MCP configuration tools.
	cfg, err := d.cm.GetConfig(ctx)
	if err != nil {
		return mcp.NewToolResultError(
			missingClusterConfigErrorFromErr(d.appName, err)), nil
	}

	// Validating the topology as a whole, dependencies and integrations to ensure
	// the cluster is ready to deploy.
	if _, err = d.topologyBuilder.Build(ctx, cfg); err != nil {
		return mcp.NewToolResultErrorFromErr(`
Ensure the cluster is properly configured and all required integrations are in
place. Inspect the error message below to assess the issue.`,
			err,
		), nil
	}

	// Deployment job flags.
	var debug, dryRun, force bool

	if v, ok := ctr.GetArguments()[DebugArg].(bool); ok {
		debug = v
	}
	if v, ok := ctr.GetArguments()[DryRunArg].(bool); ok {
		dryRun = v
	}
	if v, ok := ctr.GetArguments()[ForceArg].(bool); ok {
		force = v
	}

	// Command to get the logs of the deployment job.
	logsCmd := d.job.GetJobLogFollowCmd(cfg.Namespace())

	// Issue the deployment job using the informed flags.
	err = d.job.Run(ctx, debug, dryRun, force, cfg.Namespace(), d.image)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf(`
Unable to issue the deployment Job, it returned the following error:

> %s

In case the job exists in the cluster, use the force flag to force it's
recreation, and use the following command to inspect its current state:

	%s`,
			err.Error(), logsCmd),
		), nil
	}

	return mcp.NewToolResultText(fmt.Sprintf(`
ATTENTION: The "dry-run" flag will prevent any changes from being made to the
cluster, set the flag to "false" in order to apply changes.

The installer job has been created successfully. Use the tool %q to check the
deployment status; this can take a few minutes depending on cluster performance.
Use the tool periodically to verify that the deployment is proceeding as expected.

Informed flags:
	- debug: %v
	- dry-run: %v
	- force: %v

You can follow the Kubernetes Job logs by running:

	%s`,
		d.appName+statusSuffix, debug, dryRun, force, logsCmd,
	)), nil
}

// Init registers the deployment tools on the MCP server.
func (d *DeployTools) Init(mcpServer *server.MCPServer) {
	mcpServer.AddTools([]server.ServerTool{{
		Tool: mcp.NewTool(
			d.appName+deploySuffix,
			mcp.WithDescription(fmt.Sprintf(`
Deploys %s components to the cluster, using the cluster configuration to deploy
the components sequentially. Note the "dry-run" flag: the deployment process will
only be initiated when the "dry-run" flag is set to "false". By default, this flag
is set to "true".`,
				d.appName,
			)),
			mcp.WithBoolean(
				DryRunArg,
				mcp.Description(fmt.Sprintf(`
Run the installation Job in dry-run mode. This will not create any resources in
the cluster and is useful for validating the configuration and integrations. You
must set it to "false" in order to deploy the %s platform in your cluster.`,
					d.appName,
				)),
				mcp.DefaultBool(true),
			),
			mcp.WithBoolean(
				ForceArg,
				mcp.Description(`
Forces the recreation of the installation Job. This will delete the existing
Job and create a new one regardless of its state.`,
				),
				mcp.DefaultBool(false),
			),
			mcp.WithBoolean(
				DebugArg,
				mcp.Description(`
Sets the debug mode for the deployment job. This will enable verbose logging and
additional platform deployment information.`,
				),
				mcp.DefaultBool(false),
			),
		),
		Handler: d.deployHandler,
	}}...)
}

// NewDeployTools creates a new DeployTools instance.
func NewDeployTools(
	appName string,
	cm *config.ConfigMapManager,
	topologyBuilder *resolver.TopologyBuilder,
	job *installer.Job,
	image string,
) *DeployTools {
	return &DeployTools{appName: appName, cm: cm, topologyBuilder: topologyBuilder, job: job, image: image}
}

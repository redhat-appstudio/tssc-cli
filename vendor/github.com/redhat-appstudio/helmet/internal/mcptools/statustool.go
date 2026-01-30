package mcptools

import (
	"context"
	"errors"
	"fmt"

	"github.com/redhat-appstudio/helmet/internal/config"
	"github.com/redhat-appstudio/helmet/internal/installer"
	"github.com/redhat-appstudio/helmet/internal/resolver"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// StatusTool represents the MCP tool that's responsible to report the current
// installer status in the cluster.
type StatusTool struct {
	appName string                    // application name
	cm      *config.ConfigMapManager  // cluster configuration
	tb      *resolver.TopologyBuilder // topology builder
	job     *installer.Job            // cluster deployment job
}

var _ Interface = &StatusTool{}

const (
	// statusSuffix MCP status tool name suffix.
	statusSuffix = "_status"

	// AwaitingConfigurationPhase first step, the cluster is not configured yet.
	AwaitingConfigurationPhase = "AWAITING_CONFIGURATION"
	// AwaitingIntegrationsPhase second step, the cluster doesn't have the
	// required integrations configured yet.
	AwaitingIntegrationsPhase = "AWAITING_INTEGRATIONS"
	// ReadyToDeployPhase third step, the cluster is ready to deploy. It's
	// configured and has all required integrations in place.
	ReadyToDeployPhase = "READY_TO_DEPLOY"
	// DeployingPhase fourth step, the installer is currently deploying the
	// dependencies, Helm charts.
	DeployingPhase = "DEPLOYING"
	// CompletedPhase final step, the installation process is complete, and the
	// cluster is ready.
	CompletedPhase = "COMPLETED"
	// InstallerErrorPhase indicates an error occurred while trying to determine
	// the installer's operational status (e.g., failed to get job state).
	InstallerErrorPhase = "INSTALLER_ERROR"
)

// statusHandler shows the installer overall status by inspecting the cluster to
// determine the current state of the installation.
func (s *StatusTool) statusHandler(
	ctx context.Context,
	_ mcp.CallToolRequest,
) (*mcp.CallToolResult, error) {
	phase, err := getInstallerPhase(ctx, s.cm, s.tb, s.job)

	// Shell command to get the logs of the deployment job.
	var logsCmdEx string
	if cfg, cfgErr := s.cm.GetConfig(ctx); cfgErr == nil {
		logsCmdEx = s.job.GetJobLogFollowCmd(cfg.Namespace())
	}

	switch phase {
	case AwaitingConfigurationPhase:
		return mcp.NewToolResultText(fmt.Sprintf(
			"# Current Status: %q\n\n%s",
			phase, missingClusterConfigErrorFromErr(s.appName, err),
		)), nil
	case AwaitingIntegrationsPhase:
		switch {
		case errors.Is(err, resolver.ErrCircularDependency) ||
			errors.Is(err, resolver.ErrDependencyNotFound) ||
			errors.Is(err, resolver.ErrInvalidCollection):
			return mcp.NewToolResultText(fmt.Sprintf(`
# Current Status: %q

ATTENTION: The installer set of dependencies, Helm charts, are not properly
resolved. Please check the dependencies given to the installer. Preferably use the
embedded dependency collection.

%s`,
				phase, err.Error(),
			)), nil
		case errors.Is(err, resolver.ErrInvalidExpression) ||
			errors.Is(err, resolver.ErrUnknownIntegration):
			return mcp.NewToolResultText(fmt.Sprintf(`
# Current Status: %q

ATTENTION: The installer set of dependencies, Helm charts, are referencing invalid
required integrations expressions and/or using invalid integration names. Please
check the dependencies given to the installer. Preferably use the embedded
dependency collection.

%s`,
				phase, err.Error(),
			)), nil
		case errors.Is(err, resolver.ErrMissingIntegrations) ||
			errors.Is(err, resolver.ErrPrerequisiteIntegration):
			return mcp.NewToolResultText(fmt.Sprintf(`
# Current Status: %q

ATTENTION: One or more required integrations are missing. You must interpret the
CEL expression to help the user decide which integrations to configure. Ask the
user for input about optional integrations.

Use the tool %q to list and describe integrations, and %q to help the user
configure them.

You can use %q to verify whether the integrations are configured.

> %s`,
				phase,
				s.appName+integrationListSuffix,
				s.appName+integrationScaffoldSuffix,
				s.appName+integrationStatusSuffix,
				err.Error(),
			)), nil
		default:
			return mcp.NewToolResultError(err.Error()), nil
		}
	case ReadyToDeployPhase:
		return mcp.NewToolResultText(fmt.Sprintf(`
# Current Status: %q

The cluster is ready to deploy the %s components. Use the tool %q to deploy the
%s components.`,
			phase, s.appName, s.appName+deploySuffix, s.appName,
		)), nil
	case DeployingPhase:
		jobState, err := s.job.GetState(ctx)
		if err != nil {
			return nil, err
		}

		if jobState == installer.Failed {
			return mcp.NewToolResultText(fmt.Sprintf(`
# Current Status: %q

The deployment job has failed. You can use the following command to view the
related POD logs:

> %s`,
				phase, logsCmdEx,
			)), nil
		}

		// Assume Deploying if not Failed.
		return mcp.NewToolResultText(fmt.Sprintf(`
# Current Status: %q

The cluster is deploying the %s components. Please wait for the deployment to
complete. You can use the following command to follow the deployment job logs:

> %s`,
			phase, s.appName, logsCmdEx,
		)), nil
	case CompletedPhase:
		return mcp.NewToolResultText(fmt.Sprintf(`
# Current Status: %q

The %s components have been deployed successfully. You can use the following
command to inspect the installation logs and get initial information for each
product deployed:

> %s`,
			phase, s.appName, logsCmdEx,
		)), nil
	case InstallerErrorPhase:
		// Indicates an operational error during job state determination.
		return mcp.NewToolResultError(err.Error()), nil
	default:
		return mcp.NewToolResultError("unknown installer state"), nil
	}
}

// Init registers the status tool.
func (s *StatusTool) Init(mcpServer *server.MCPServer) {
	mcpServer.AddTools([]server.ServerTool{{
		Tool: mcp.NewTool(
			s.appName+statusSuffix,
			mcp.WithDescription(`
Reports the overall installer status, the first tool to be called to identify the
installer status in the cluster and define the next tool to call.
			`),
		),
		Handler: s.statusHandler,
	}}...)
}

// NewStatusTool creates a new StatusTool instance.
func NewStatusTool(
	appName string,
	cm *config.ConfigMapManager,
	tb *resolver.TopologyBuilder,
	job *installer.Job,
) *StatusTool {
	return &StatusTool{
		appName: appName,
		cm:      cm,
		tb:      tb,
		job:     job,
	}
}

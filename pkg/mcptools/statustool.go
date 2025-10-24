package mcptools

import (
	"context"
	"errors"
	"fmt"

	"github.com/redhat-appstudio/tssc-cli/pkg/config"
	"github.com/redhat-appstudio/tssc-cli/pkg/constants"
	"github.com/redhat-appstudio/tssc-cli/pkg/installer"
	"github.com/redhat-appstudio/tssc-cli/pkg/resolver"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// StatusTool represents the MCP tool that's responsible to report the current
// installer status in the cluster.
type StatusTool struct {
	cm              *config.ConfigMapManager  // cluster configuration
	topologyBuilder *resolver.TopologyBuilder // topology builder
	job             *installer.Job            // cluster deployment job

	phase string // current deployment phase
}

var _ Interface = &StatusTool{}

const (
	// StatusToolName MCP status tool name.
	StatusToolName = constants.AppName + "_status"

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
)

// resultWithPhaseF uses the global phase, the informed message format and
// arguments to compose the tool result. The installer status name is place before
// the informed message.
func (s *StatusTool) resultWithPhaseF(
	format string,
	a ...any,
) *mcp.CallToolResult {
	return mcp.NewToolResultText(
		fmt.Sprintf("# Current Status: %q\n", s.phase) +
			fmt.Sprintf(format, a...),
	)
}

// statusHandler shows the installer overall status by inspecting the cluster to
// determine the current state of the installation.
func (s *StatusTool) statusHandler(
	ctx context.Context,
	_ mcp.CallToolRequest,
) (*mcp.CallToolResult, error) {
	// Ensure the cluster is configured, if the ConfigMap is not found, creates a
	// message to inform the user about MCP configuration tools.
	s.phase = AwaitingConfigurationPhase
	cfg, err := s.cm.GetConfig(ctx)
	if err != nil {
		return s.resultWithPhaseF(`
The cluster is not configured yet, use the tool %q to configure it. That's the
first step to deploy TSSC components.

Inspecting the configuration in the cluster returned the following error:

> %s`,
			ConfigInitTool, err.Error(),
		), nil
	}

	// Given the cluster is configured, let's inspect the topology to ensure all
	// dependencies and integrations are resolved.
	s.phase = AwaitingIntegrationsPhase
	if _, err = s.topologyBuilder.Build(ctx, cfg); err != nil {
		switch {
		case errors.Is(err, resolver.ErrCircularDependency) ||
			errors.Is(err, resolver.ErrDependencyNotFound) ||
			errors.Is(err, resolver.ErrInvalidCollection):
			return s.resultWithPhaseF(`
ATTENTION: The installer set of dependencies, Helm charts, are not properly
resolved. Please check the dependencies given to the installer. Preferably use the
embedded dependency collection.

%s`,
				err.Error(),
			), nil
		case errors.Is(err, resolver.ErrInvalidExpression) ||
			errors.Is(err, resolver.ErrUnknownIntegration):
			return s.resultWithPhaseF(`
ATTENTION: The installer set of dependencies, Helm charts, are referencing invalid
required integrations expressions and/or using invalid integration names. Please
check the dependencies given to the installer. Preferably use the embedded
dependency collection.

%s`,
				err.Error(),
			), nil
		case errors.Is(err, resolver.ErrMissingIntegrations) ||
			errors.Is(err, resolver.ErrPrerequisiteIntegration):
			return s.resultWithPhaseF(`
ATTENTION: One or more required integrations are missing. You must interpret the
CEL expression to help the user decide which integrations to configure. Ask the
user for input about optional integrations.

Use the tool %q to list and describe integrations, and %q to help the user
configure them.

You can use %q to verify whether the integrations are configured.

> %s`,
				IntegrationListTool,
				IntegrationScaffoldTool,
				IntegrationStatusTool,
				err.Error(),
			), nil
		default:
			return mcp.NewToolResultError(err.Error()), nil
		}
	}

	// Given integrations are in place, let's inspect the current state of the
	// cluster deployment job.
	jobState, err := s.job.GetState(ctx)
	if err != nil {
		return nil, err
	}
	// Shell command to get the logs of the deployment job.
	logsCmdEx := s.job.GetJobLogFollowCmd(cfg.Installer.Namespace)

	// Handle different states of the deployment job.
	switch jobState {
	case installer.NotFound:
		s.phase = ReadyToDeployPhase
		return s.resultWithPhaseF(`
The cluster is ready to deploy the TSSC components. Use the tool %q to deploy the
TSSC components.`,
			DeployToolName,
		), nil
	case installer.Deploying:
		s.phase = DeployingPhase
		return s.resultWithPhaseF(`
The cluster is deploying the TSSC components. Please wait for the deployment to
complete. You can use the following command to follow the deployment job logs:

> %s`,
			logsCmdEx,
		), nil
	case installer.Failed:
		s.phase = DeployingPhase
		return s.resultWithPhaseF(`
The deployment job has failed. You can use the following command to view the
related POD logs:

> %s`,
			logsCmdEx,
		), nil
	case installer.Done:
		s.phase = CompletedPhase
		return s.resultWithPhaseF(`
The TSSC components have been deployed successfully. You can use the following
command to inspect the installation logs and get initial information for each
product deployed:

> %s`,
			logsCmdEx,
		), nil
	}

	return mcp.NewToolResultError("unknown installer state"), nil
}

// Init registers the status tool.
func (s *StatusTool) Init(mcpServer *server.MCPServer) {
	mcpServer.AddTools([]server.ServerTool{{
		Tool: mcp.NewTool(
			StatusToolName,
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
	cm *config.ConfigMapManager,
	tb *resolver.TopologyBuilder,
	job *installer.Job,
) *StatusTool {
	return &StatusTool{
		cm:              cm,
		topologyBuilder: tb,
		job:             job,
	}
}

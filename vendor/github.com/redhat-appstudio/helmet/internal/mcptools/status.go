package mcptools

import (
	"context"
	"errors"

	"github.com/redhat-appstudio/helmet/internal/config"
	"github.com/redhat-appstudio/helmet/internal/installer"
	"github.com/redhat-appstudio/helmet/internal/resolver"
)

func getInstallerPhase(
	ctx context.Context,
	cm *config.ConfigMapManager,
	tb *resolver.TopologyBuilder,
	job *installer.Job,
) (string, error) {
	// Ensure the cluster is configured.
	cfg, err := cm.GetConfig(ctx)
	if err != nil {
		// If config is missing, we are in AwaitingConfigurationPhase.
		// The specific error will be used by the caller for detailed messaging.
		return AwaitingConfigurationPhase, err
	}

	// Given the cluster is configured, inspect the topology to ensure all
	// dependencies and integrations are resolved.
	if _, err = tb.Build(ctx, cfg); err != nil {
		// If topology build fails, we are in AwaitingIntegrationsPhase.
		// The specific resolver error will be used by the caller for detailed messaging.
		return AwaitingIntegrationsPhase, err
	}

	// Given integrations are in place, inspect the current state of the
	// cluster deployment job.
	jobState, err := job.GetState(ctx)
	if err != nil {
		// If job state cannot be determined, it's an operational error.
		// Return InstallerErrorPhase with the original error.
		return InstallerErrorPhase, err
	}

	// Map the job state to an installer phase.
	switch jobState {
	case installer.NotFound:
		return ReadyToDeployPhase, nil
	case installer.Deploying, installer.Failed:
		// Both 'Deploying' and 'Failed' states indicate that the deployment
		// process is active or has attempted to run, thus falling under
		// the 'DeployingPhase' for overall status reporting.
		return DeployingPhase, nil
	case installer.Done:
		return CompletedPhase, nil
	default:
		// Unrecognized installer state from s.job.GetState.
		// This is also an operational error.
		return InstallerErrorPhase,
			errors.New("unknown installer job state reported by cluster")
	}
}

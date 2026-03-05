package subcmd

import (
	"fmt"

	"github.com/redhat-appstudio/helmet/api"
	"github.com/redhat-appstudio/helmet/internal/config"
	"github.com/redhat-appstudio/helmet/internal/integration"
	"github.com/redhat-appstudio/helmet/internal/runcontext"

	"github.com/spf13/cobra"
)

// IntegrationArtifactory is the sub-command for the "integration artifactory",
// responsible for creating and updating the Artifactory integration secret.
type IntegrationArtifactory struct {
	cmd         *cobra.Command           // cobra command
	appCtx      *api.AppContext          // application context
	runCtx      *runcontext.RunContext   // run context (kube, logger, chartfs)
	cfg         *config.Config           // installer configuration
	integration *integration.Integration // integration instance
}

var _ api.SubCommand = &IntegrationArtifactory{}

// Cmd exposes the cobra instance.
func (a *IntegrationArtifactory) Cmd() *cobra.Command {
	return a.cmd
}

// Complete is a no-op in this case.
func (a *IntegrationArtifactory) Complete(_ []string) error {
	var err error
	a.cfg, err = bootstrapConfig(a.cmd.Context(), a.appCtx, a.runCtx)
	return err
}

// Validate checks if the required configuration is set.
func (a *IntegrationArtifactory) Validate() error {
	return a.integration.Validate()
}

// Run creates or updates the Artifactory integration secret.
func (a *IntegrationArtifactory) Run() error {
	return a.integration.Create(a.cmd.Context(), a.runCtx, a.cfg)
}

// NewIntegrationArtifactory creates the sub-command for the "integration artifactory"
// responsible to manage the integrations with a Artifactory image registry.
func NewIntegrationArtifactory(
	appCtx *api.AppContext,
	runCtx *runcontext.RunContext,
	i *integration.Integration,
) *IntegrationArtifactory {
	a := &IntegrationArtifactory{
		cmd: &cobra.Command{
			Use: "artifactory [flags]",
			Short: fmt.Sprintf(
				"Integrates an Artifactory instance into %s",
				appCtx.Name,
			),
			Long: fmt.Sprintf(`
Manages the Artifactory integration with %s by storing the credentials
required by %s services to interact with Artifactory.

The credentials are stored in a Kubernetes Secret in the namespace
configured for %s.`,
				appCtx.Name,
				appCtx.Name,
				appCtx.Name,
			),
			SilenceUsage: true,
		},

		appCtx:      appCtx,
		runCtx:      runCtx,
		integration: i,
	}
	i.PersistentFlags(a.cmd)
	return a
}

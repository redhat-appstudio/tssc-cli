package subcmd

import (
	"fmt"

	"github.com/redhat-appstudio/helmet/api"
	"github.com/redhat-appstudio/helmet/internal/config"
	"github.com/redhat-appstudio/helmet/internal/integration"
	"github.com/redhat-appstudio/helmet/internal/runcontext"

	"github.com/spf13/cobra"
)

// IntegrationTrustedArtifactSigner is the sub-command for the "integration trusted-artifact-signer",
// responsible for creating and updating the TrustedArtifactSigner integration secret.
type IntegrationTrustedArtifactSigner struct {
	cmd         *cobra.Command           // cobra command
	appCtx      *api.AppContext          // application context
	runCtx      *runcontext.RunContext   // run context (kube, logger, chartfs)
	cfg         *config.Config           // installer configuration
	integration *integration.Integration // integration instance
}

var _ api.SubCommand = &IntegrationTrustedArtifactSigner{}

// Cmd exposes the cobra instance.
func (t *IntegrationTrustedArtifactSigner) Cmd() *cobra.Command {
	return t.cmd
}

// Complete is a no-op in this case.
func (t *IntegrationTrustedArtifactSigner) Complete(_ []string) error {
	var err error
	t.cfg, err = bootstrapConfig(t.cmd.Context(), t.appCtx, t.runCtx)
	return err
}

// Validate checks if the required configuration is set.
func (t *IntegrationTrustedArtifactSigner) Validate() error {
	return t.integration.Validate()
}

// Run creates or updates the TrustedArtifactSigner integration secret.
func (t *IntegrationTrustedArtifactSigner) Run() error {
	return t.integration.Create(t.cmd.Context(), t.runCtx, t.cfg)
}

// NewIntegrationTrustedArtifactSigner creates the sub-command for the "integration
// trusted-artifact-signer" responsible to manage the integrations with the
// Trusted Artifact Signer services.
func NewIntegrationTrustedArtifactSigner(
	appCtx *api.AppContext,
	runCtx *runcontext.RunContext,
	i *integration.Integration,
) *IntegrationTrustedArtifactSigner {
	t := &IntegrationTrustedArtifactSigner{
		cmd: &cobra.Command{
			Use: "trusted-artifact-signer [flags]",
			Short: fmt.Sprintf(
				"Integrates a Trusted Artifact Signer instance into %s",
				appCtx.Name,
			),
			Long: fmt.Sprintf(`
Manages the Trusted Artifact Signer integration with %s by storing the
URL required to interact with the Trusted Artifact Signer service.

The configuration is stored in a Kubernetes Secret in the namespace
configured for %s.`,
				appCtx.Name,
				appCtx.Name,
			),
			SilenceUsage: true,
		},

		appCtx:      appCtx,
		runCtx:      runCtx,
		integration: i,
	}
	i.PersistentFlags(t.cmd)
	return t
}

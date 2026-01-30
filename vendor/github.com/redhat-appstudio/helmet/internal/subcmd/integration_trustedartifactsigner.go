package subcmd

import (
	"log/slog"

	"github.com/redhat-appstudio/helmet/api"

	"github.com/redhat-appstudio/helmet/internal/config"
	"github.com/redhat-appstudio/helmet/internal/integration"
	"github.com/redhat-appstudio/helmet/internal/k8s"

	"github.com/spf13/cobra"
)

// IntegrationTrustedArtifactSigner is the sub-command for the "integration trusted-artifact-signer",
// responsible for creating and updating the TrustedArtifactSigner integration secret.
type IntegrationTrustedArtifactSigner struct {
	cmd         *cobra.Command           // cobra command
	appCtx      *api.AppContext          // application context
	logger      *slog.Logger             // application logger
	cfg         *config.Config           // installer configuration
	kube        *k8s.Kube                // kubernetes client
	integration *integration.Integration // integration instance
}

var _ api.SubCommand = &IntegrationTrustedArtifactSigner{}

const trustedArtifactSignerIntegrationLongDesc = `
Manages the TrustedArtifactSigner integration with TSSC, by storing the required
URL required to interact with Trusted Artifact Signer.

The credentials are stored in a Kubernetes Secret in the configured namespace for TSSC.`

// Cmd exposes the cobra instance.
func (t *IntegrationTrustedArtifactSigner) Cmd() *cobra.Command {
	return t.cmd
}

// Complete is a no-op in this case.
func (t *IntegrationTrustedArtifactSigner) Complete(args []string) error {
	var err error
	t.cfg, err = bootstrapConfig(t.cmd.Context(), t.appCtx, t.kube)
	return err
}

// Validate checks if the required configuration is set.
func (t *IntegrationTrustedArtifactSigner) Validate() error {
	return t.integration.Validate()
}

// Run creates or updates the TrustedArtifactSigner integration secret.
func (t *IntegrationTrustedArtifactSigner) Run() error {
	return t.integration.Create(t.cmd.Context(), t.cfg)
}

// NewIntegrationTrustedArtifactSigner creates the sub-command for the "integration
// trusted-artifact-signer" responsible to manage the TSSC integrations with the
// Trusted Artifact Signer services.
func NewIntegrationTrustedArtifactSigner(
	appCtx *api.AppContext,
	logger *slog.Logger,
	kube *k8s.Kube,
	i *integration.Integration,
) *IntegrationTrustedArtifactSigner {
	t := &IntegrationTrustedArtifactSigner{
		cmd: &cobra.Command{
			Use:          "trusted-artifact-signer [flags]",
			Short:        "Integrates a Trusted Artifact Signer instance into TSSC",
			Long:         trustedArtifactSignerIntegrationLongDesc,
			SilenceUsage: true,
		},

		appCtx:      appCtx,
		logger:      logger,
		kube:        kube,
		integration: i,
	}
	i.PersistentFlags(t.cmd)
	return t
}

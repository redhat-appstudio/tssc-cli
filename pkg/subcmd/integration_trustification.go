package subcmd

import (
	"log/slog"

	"github.com/redhat-appstudio/rhtap-cli/pkg/config"
	"github.com/redhat-appstudio/rhtap-cli/pkg/integrations"
	"github.com/redhat-appstudio/rhtap-cli/pkg/k8s"

	"github.com/spf13/cobra"
)

// IntegrationTrustification is the sub-command for the "integration trustification",
// responsible for creating and updating the Trustification integration secret.
type IntegrationTrustification struct {
	cmd    *cobra.Command // cobra command
	logger *slog.Logger   // application logger
	cfg    *config.Config // installer configuration
	kube   *k8s.Kube      // kubernetes client

	trustificationIntegration *integrations.TrustificationIntegration // trustification integration

	bombasticAPIURL           string // URL of the BOMbastic api host
	oidcIssuerURL             string // URL of the OIDC token issuer
	oidcClientId              string // OIDC client ID
	oidcClientSecret          string // OIDC client secret
	supportedCyclonedxVersion string // If specified the SBOM will be converted to the supported version before uploading.
}

var _ Interface = &IntegrationTrustification{}

const trustificationIntegrationLongDesc = `
Manages the Trustification integration with TSSC, by storing the required
credentials required by the TSSC services to interact with Trustification.

The credentials are stored in a Kubernetes Secret in the configured namespace
for RHDH.
`

// Cmd exposes the cobra instance.
func (d *IntegrationTrustification) Cmd() *cobra.Command {
	return d.cmd
}

// Complete is a no-op in this case.
func (d *IntegrationTrustification) Complete(args []string) error {
	var err error
	d.cfg, err = bootstrapConfig(d.cmd.Context(), d.kube)
	return err
}

// Validate checks if the required configuration is set.
func (d *IntegrationTrustification) Validate() error {
	return d.trustificationIntegration.Validate()
}

// Run creates or updates the Trustification integration secret.
func (d *IntegrationTrustification) Run() error {
	err := d.trustificationIntegration.EnsureNamespace(d.cmd.Context(), d.cfg)
	if err != nil {
		return err
	}
	return d.trustificationIntegration.Create(d.cmd.Context(), d.cfg)
}

// NewIntegrationTrustification creates the sub-command for the "integration
// trustification" responsible to manage the TSSC integrations with the
// Trustification service.
func NewIntegrationTrustification(
	logger *slog.Logger,
	kube *k8s.Kube,
) *IntegrationTrustification {
	trustificationIntegration := integrations.NewTrustificationIntegration(logger, kube)

	d := &IntegrationTrustification{
		cmd: &cobra.Command{
			Use:          "trustification [flags]",
			Short:        "Integrates a Trustification instance into TSSC",
			Long:         trustificationIntegrationLongDesc,
			SilenceUsage: true,
		},

		logger: logger,
		kube:   kube,

		trustificationIntegration: trustificationIntegration,
	}

	p := d.cmd.PersistentFlags()
	trustificationIntegration.PersistentFlags(p)
	return d
}

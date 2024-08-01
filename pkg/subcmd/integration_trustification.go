package subcmd

import (
	"fmt"
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
Manages the Trustification integration with RHTAP, by storing the required
credentials required by the RHTAP services to interact with Trustification.

The credentials are stored in a Kubernetes Secret in the configured namespace
for RHDH.
`

// Cmd exposes the cobra instance.
func (d *IntegrationTrustification) Cmd() *cobra.Command {
	return d.cmd
}

// Complete is a no-op in this case.
func (d *IntegrationTrustification) Complete(args []string) error {
	return nil
}

// Validate checks if the required configuration is set.
func (d *IntegrationTrustification) Validate() error {
	feature, err := d.cfg.GetFeature(config.RedHatDeveloperHub)
	if err != nil {
		return err
	}
	if !feature.Enabled {
		return fmt.Errorf("Red Hat Developer Hub feature is not enabled")
	}
	return d.trustificationIntegration.Validate()
}

// Run creates or updates the Trustification integration secret.
func (d *IntegrationTrustification) Run() error {
	if err := d.trustificationIntegration.EnsureNamespace(d.cmd.Context()); err != nil {
		return err
	}
	return d.trustificationIntegration.Create(d.cmd.Context())
}

// NewIntegrationTrustification creates the sub-command for the "integration trustification"
// responsible to manage the RHTAP integrations with the Trustification service.
func NewIntegrationTrustification(
	logger *slog.Logger,
	cfg *config.Config,
	kube *k8s.Kube,
) *IntegrationTrustification {
	trustificationIntegration := integrations.NewTrustificationIntegration(logger, cfg, kube)

	d := &IntegrationTrustification{
		cmd: &cobra.Command{
			Use:          "trustification [flags]",
			Short:        "Integrates a Trustification instance into RHTAP",
			Long:         trustificationIntegrationLongDesc,
			SilenceUsage: true,
		},

		logger: logger,
		cfg:    cfg,
		kube:   kube,

		trustificationIntegration: trustificationIntegration,
	}

	p := d.cmd.PersistentFlags()
	trustificationIntegration.PersistentFlags(p)
	return d
}

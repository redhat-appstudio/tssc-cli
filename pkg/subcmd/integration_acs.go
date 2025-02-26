package subcmd

import (
	"fmt"
	"log/slog"

	"github.com/redhat-appstudio/rhtap-cli/pkg/config"
	"github.com/redhat-appstudio/rhtap-cli/pkg/integrations"
	"github.com/redhat-appstudio/rhtap-cli/pkg/k8s"

	"github.com/spf13/cobra"
)

// IntegrationACS is the sub-command for the "integration acs",
// responsible for creating and updating the ACS integration secret.
type IntegrationACS struct {
	cmd    *cobra.Command // cobra command
	logger *slog.Logger   // application logger
	cfg    *config.Config // installer configuration
	kube   *k8s.Kube      // kubernetes client

	acsIntegration *integrations.ACSIntegration // acs integration

	apiToken string // API token
	endpoint string // service endpoint
}

var _ Interface = &IntegrationACS{}

const acsIntegrationLongDesc = `
Manages the ACS integration with RHTAP, by storing the required
credentials required by the RHTAP services to interact with ACS.

The credentials are stored in a Kubernetes Secret in the configured namespace
for RHDH.
`

// Cmd exposes the cobra instance.
func (d *IntegrationACS) Cmd() *cobra.Command {
	return d.cmd
}

// Complete loads the configuration from cluster.
func (d *IntegrationACS) Complete(args []string) error {
	var err error
	d.cfg, err = bootstrapConfig(d.cmd.Context(), d.kube)
	return err
}

// Validate checks if the required configuration is set.
func (d *IntegrationACS) Validate() error {
	feature, err := d.cfg.GetFeature(config.RedHatDeveloperHub)
	if err != nil {
		return err
	}
	if !feature.Enabled {
		return fmt.Errorf("Red Hat Developer Hub feature is not enabled")
	}
	return d.acsIntegration.Validate()
}

// Run creates or updates the ACS integration secret.
func (d *IntegrationACS) Run() error {
	if err := d.acsIntegration.EnsureNamespace(d.cmd.Context(), d.cfg); err != nil {
		return err
	}
	return d.acsIntegration.Create(d.cmd.Context(), d.cfg)
}

// NewIntegrationACS creates the sub-command for the "integration acs"
// responsible to manage the RHTAP integrations with the ACS service.
func NewIntegrationACS(
	logger *slog.Logger,
	kube *k8s.Kube,
) *IntegrationACS {
	acsIntegration := integrations.NewACSIntegration(logger, kube)

	d := &IntegrationACS{
		cmd: &cobra.Command{
			Use:          "acs [flags]",
			Short:        "Integrates a ACS instance into RHTAP",
			Long:         acsIntegrationLongDesc,
			SilenceUsage: true,
		},

		logger: logger,
		kube:   kube,

		acsIntegration: acsIntegration,
	}

	p := d.cmd.PersistentFlags()
	acsIntegration.PersistentFlags(p)
	return d
}

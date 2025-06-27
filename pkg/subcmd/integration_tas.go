package subcmd

import (
	"log/slog"

	"github.com/redhat-appstudio/rhtap-cli/pkg/config"
	"github.com/redhat-appstudio/rhtap-cli/pkg/integrations"
	"github.com/redhat-appstudio/rhtap-cli/pkg/k8s"

	"github.com/spf13/cobra"
)

// IntegrationTAS is the sub-command for the "integration tas",
// responsible for creating and updating the tas integration secret.
type IntegrationTAS struct {
	cmd    *cobra.Command // cobra command
	logger *slog.Logger   // application logger
	cfg    *config.Config // installer configuration
	kube   *k8s.Kube      // kubernetes client

	tasIntegration *integrations.TASIntegration // tas integration

	rekorURL string // rekor url
	tufURL   string // tuf url
}

var _ Interface = &IntegrationTAS{}

const tasIntegrationLongDesc = `
Manages the TAS integration with TSSC, by storing the required
credentials required by the TSSC services to interact with TAS.

The credentials are stored in a Kubernetes Secret in the configured namespace
for RHDH.
`

// Cmd exposes the cobra instance.
func (d *IntegrationTAS) Cmd() *cobra.Command {
	return d.cmd
}

// Complete loads the configuration from cluster.
func (d *IntegrationTAS) Complete(args []string) error {
	var err error
	d.cfg, err = bootstrapConfig(d.cmd.Context(), d.kube)
	return err
}

// Validate checks if the required configuration is set.
func (d *IntegrationTAS) Validate() error {
	return d.tasIntegration.Validate()
}

// Run creates or updates the TAS integration secret.
func (d *IntegrationTAS) Run() error {
	if err := d.tasIntegration.EnsureNamespace(d.cmd.Context(), d.cfg); err != nil {
		return err
	}
	return d.tasIntegration.Create(d.cmd.Context(), d.cfg)
}

// NewIntegrationTAS creates the sub-command for the "integration tas"
// responsible to manage the TSSC integrations with the TAS service.
func NewIntegrationTAS(
	logger *slog.Logger,
	kube *k8s.Kube,
) *IntegrationTAS {
	tasIntegration := integrations.NewTASIntegration(logger, kube)

	d := &IntegrationTAS{
		cmd: &cobra.Command{
			Use:          "tas [flags]",
			Short:        "Integrates a TAS instance into TSSC",
			Long:         tasIntegrationLongDesc,
			SilenceUsage: true,
		},

		logger: logger,
		kube:   kube,

		tasIntegration: tasIntegration,
	}

	p := d.cmd.PersistentFlags()
	tasIntegration.PersistentFlags(p)
	return d
}

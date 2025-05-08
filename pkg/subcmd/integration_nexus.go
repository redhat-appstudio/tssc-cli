package subcmd

import (
	"log/slog"

	"github.com/redhat-appstudio/rhtap-cli/pkg/config"
	"github.com/redhat-appstudio/rhtap-cli/pkg/integrations"
	"github.com/redhat-appstudio/rhtap-cli/pkg/k8s"

	"github.com/spf13/cobra"
)

// IntegrationNexus is the sub-command for the "integration nexus",
// responsible for creating and updating the Nexus integration secret.
type IntegrationNexus struct {
	cmd    *cobra.Command // cobra command
	logger *slog.Logger   // application logger
	cfg    *config.Config // installer configuration
	kube   *k8s.Kube      // kubernetes client

	nexusIntegration *integrations.NexusIntegration // nexus integration

	dockerconfigjson string // credentials to push/pull from the registry
}

var _ Interface = &IntegrationNexus{}

const nexusIntegrationLongDesc = `
Manages the Nexus integration with TSSC, by storing the required
credentials required by the TSSC services to interact with Nexus.

The credentials are stored in a Kubernetes Secret in the configured namespace
for RHDH.
`

// Cmd exposes the cobra instance.
func (d *IntegrationNexus) Cmd() *cobra.Command {
	return d.cmd
}

// Complete is a no-op in this case.
func (d *IntegrationNexus) Complete(args []string) error {
	var err error
	d.cfg, err = bootstrapConfig(d.cmd.Context(), d.kube)
	return err
}

// Validate checks if the required configuration is set.
func (d *IntegrationNexus) Validate() error {
	return d.nexusIntegration.Validate()
}

// Run creates or updates the Nexus integration secret.
func (d *IntegrationNexus) Run() error {
	err := d.nexusIntegration.EnsureNamespace(d.cmd.Context(), d.cfg)
	if err != nil {
		return err
	}
	return d.nexusIntegration.Create(d.cmd.Context(), d.cfg)
}

// NewIntegrationNexus creates the sub-command for the "integration nexus"
// responsible to manage the TSSC integrations with a Nexus image registry.
func NewIntegrationNexus(
	logger *slog.Logger,
	kube *k8s.Kube,
) *IntegrationNexus {
	nexusIntegration := integrations.NewNexusIntegration(logger, kube)

	d := &IntegrationNexus{
		cmd: &cobra.Command{
			Use:          "nexus [flags]",
			Short:        "Integrates a Nexus instance into TSSC",
			Long:         nexusIntegrationLongDesc,
			SilenceUsage: true,
		},

		logger: logger,
		kube:   kube,

		nexusIntegration: nexusIntegration,
	}

	p := d.cmd.PersistentFlags()
	nexusIntegration.PersistentFlags(p)
	return d
}

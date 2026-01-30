package subcmd

import (
	"log/slog"

	"github.com/redhat-appstudio/helmet/api"

	"github.com/redhat-appstudio/helmet/internal/config"
	"github.com/redhat-appstudio/helmet/internal/integration"
	"github.com/redhat-appstudio/helmet/internal/k8s"

	"github.com/spf13/cobra"
)

// IntegrationNexus is the sub-command for the "integration nexus",
// responsible for creating and updating the Nexus integration secret.
type IntegrationNexus struct {
	cmd         *cobra.Command           // cobra command
	appCtx      *api.AppContext          // application context
	logger      *slog.Logger             // application logger
	cfg         *config.Config           // installer configuration
	kube        *k8s.Kube                // kubernetes client
	integration *integration.Integration // integration instance
}

var _ api.SubCommand = &IntegrationNexus{}

const nexusIntegrationLongDesc = `
Manages the Nexus integration with TSSC, by storing the required
credentials required by the TSSC services to interact with Nexus.

The credentials are stored in a Kubernetes Secret in the configured namespace
for RHDH.
`

// Cmd exposes the cobra instance.
func (n *IntegrationNexus) Cmd() *cobra.Command {
	return n.cmd
}

// Complete is a no-op in this case.
func (n *IntegrationNexus) Complete(args []string) error {
	var err error
	n.cfg, err = bootstrapConfig(n.cmd.Context(), n.appCtx, n.kube)
	return err
}

// Validate checks if the required configuration is set.
func (n *IntegrationNexus) Validate() error {
	return n.integration.Validate()
}

// Run creates or updates the Nexus integration secret.
func (n *IntegrationNexus) Run() error {
	return n.integration.Create(n.cmd.Context(), n.cfg)
}

// NewIntegrationNexus creates the sub-command for the "integration nexus"
// responsible to manage the TSSC integrations with a Nexus image registry.
func NewIntegrationNexus(
	appCtx *api.AppContext,
	logger *slog.Logger,
	kube *k8s.Kube,
	i *integration.Integration,
) *IntegrationNexus {
	n := &IntegrationNexus{
		cmd: &cobra.Command{
			Use:          "nexus [flags]",
			Short:        "Integrates a Nexus instance into TSSC",
			Long:         nexusIntegrationLongDesc,
			SilenceUsage: true,
		},

		appCtx:      appCtx,
		logger:      logger,
		kube:        kube,
		integration: i,
	}
	i.PersistentFlags(n.cmd)
	return n
}

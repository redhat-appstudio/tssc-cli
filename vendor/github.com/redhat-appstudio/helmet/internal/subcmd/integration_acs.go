package subcmd

import (
	"log/slog"

	"github.com/redhat-appstudio/helmet/api"
	"github.com/redhat-appstudio/helmet/internal/config"
	"github.com/redhat-appstudio/helmet/internal/integration"
	"github.com/redhat-appstudio/helmet/internal/k8s"

	"github.com/spf13/cobra"
)

// IntegrationACS is the sub-command for the "integration acs",
// responsible for creating and updating the ACS integration secret.
type IntegrationACS struct {
	cmd         *cobra.Command           // cobra command
	appCtx      *api.AppContext          // application context
	logger      *slog.Logger             // application logger
	cfg         *config.Config           // installer configuration
	kube        *k8s.Kube                // kubernetes client
	integration *integration.Integration // integration instance
}

var _ api.SubCommand = &IntegrationACS{}

const acsIntegrationLongDesc = `
Manages the ACS integration with TSSC, by storing the required
credentials required by the TSSC services to interact with ACS.

The credentials are stored in a Kubernetes Secret in the configured namespace
for RHDH.
`

// Cmd exposes the cobra instance.
func (a *IntegrationACS) Cmd() *cobra.Command {
	return a.cmd
}

// Complete loads the configuration from cluster.
func (a *IntegrationACS) Complete(args []string) error {
	var err error
	a.cfg, err = bootstrapConfig(a.cmd.Context(), a.appCtx, a.kube)
	return err
}

// Validate checks if the required configuration is set.
func (a *IntegrationACS) Validate() error {
	return a.integration.Validate()
}

// Run creates or updates the ACS integration secret.
func (a *IntegrationACS) Run() error {
	return a.integration.Create(a.cmd.Context(), a.cfg)
}

// NewIntegrationACS creates the sub-command for the "integration acs"
// responsible to manage the TSSC integrations with the ACS service.
func NewIntegrationACS(
	appCtx *api.AppContext,
	logger *slog.Logger,
	kube *k8s.Kube,
	i *integration.Integration,
) *IntegrationACS {
	a := &IntegrationACS{
		cmd: &cobra.Command{
			Use:          "acs [flags]",
			Short:        "Integrates a ACS instance into TSSC",
			Long:         acsIntegrationLongDesc,
			SilenceUsage: true,
		},

		appCtx:      appCtx,
		logger:      logger,
		kube:        kube,
		integration: i,
	}
	i.PersistentFlags(a.cmd)
	return a
}

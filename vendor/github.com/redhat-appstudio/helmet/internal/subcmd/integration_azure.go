package subcmd

import (
	"log/slog"

	"github.com/redhat-appstudio/helmet/api"

	"github.com/redhat-appstudio/helmet/internal/config"
	"github.com/redhat-appstudio/helmet/internal/integration"
	"github.com/redhat-appstudio/helmet/internal/k8s"

	"github.com/spf13/cobra"
)

// IntegrationAzure is the sub-command for the "integration azure",
// responsible for creating and updating the Azure integration secret.
type IntegrationAzure struct {
	cmd         *cobra.Command           // cobra command
	appCtx      *api.AppContext          // application context
	logger      *slog.Logger             // application logger
	cfg         *config.Config           // installer configuration
	kube        *k8s.Kube                // kubernetes client
	integration *integration.Integration // integration instance
}

var _ api.SubCommand = &IntegrationAzure{}

const azureIntegrationLongDesc = `
Manages the Azure integration with TSSC, by storing the required
credentials required by the TSSC services to interact with Azure.
The credentials are stored in a Kubernetes Secret in the default
installation namespace.
`

// Cmd exposes the cobra instance.
func (a *IntegrationAzure) Cmd() *cobra.Command {
	return a.cmd
}

// Complete is a no-op in this case.
func (a *IntegrationAzure) Complete(args []string) error {
	var err error
	a.cfg, err = bootstrapConfig(a.cmd.Context(), a.appCtx, a.kube)
	return err
}

// Validate checks if the required configuration is set.
func (a *IntegrationAzure) Validate() error {
	return a.integration.Validate()
}

// Run creates or updates the Azure integration secret.
func (a *IntegrationAzure) Run() error {
	return a.integration.Create(a.cmd.Context(), a.cfg)
}

// NewIntegrationAzure creates the sub-command for the "integration azure"
// responsible to manage the TSSC integrations with the Azure service.
func NewIntegrationAzure(
	appCtx *api.AppContext,
	logger *slog.Logger,
	kube *k8s.Kube,
	i *integration.Integration,
) *IntegrationAzure {
	a := &IntegrationAzure{
		cmd: &cobra.Command{
			Use:          "azure [flags]",
			Short:        "Integrates a Azure instance into TSSC",
			Long:         azureIntegrationLongDesc,
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

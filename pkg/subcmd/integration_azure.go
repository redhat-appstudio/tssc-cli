package subcmd

import (
	"log/slog"

	"github.com/redhat-appstudio/tssc/pkg/config"
	"github.com/redhat-appstudio/tssc/pkg/integrations"
	"github.com/redhat-appstudio/tssc/pkg/k8s"

	"github.com/spf13/cobra"
)

// IntegrationAzure is the sub-command for the "integration azure",
// responsible for creating and updating the Azure integration secret.
type IntegrationAzure struct {
	cmd    *cobra.Command // cobra command
	logger *slog.Logger   // application logger
	cfg    *config.Config // installer configuration
	kube   *k8s.Kube      // kubernetes client

	azureIntegration *integrations.AzureIntegration // azure integration

	host         string // E.g. 'dev.azure.com'
	token        string // API token
	org          string // Organization name
	clientId     string // Client ID
	clientSecret string // Client Secret
	tenantId     string // tenant ID
}

var _ Interface = &IntegrationAzure{}

const azureIntegrationLongDesc = `
Manages the Azure integration with TSSC, by storing the required
credentials required by the TSSC services to interact with Azure.
The credentials are stored in a Kubernetes Secret in the default
installation namespace.
`

// Cmd exposes the cobra instance.
func (d *IntegrationAzure) Cmd() *cobra.Command {
	return d.cmd
}

// Complete is a no-op in this case.
func (d *IntegrationAzure) Complete(args []string) error {
	var err error
	d.cfg, err = bootstrapConfig(d.cmd.Context(), d.kube)
	return err
}

// Validate checks if the required configuration is set.
func (d *IntegrationAzure) Validate() error {
	return d.azureIntegration.Validate()
}

// Run creates or updates the Azure integration secret.
func (d *IntegrationAzure) Run() error {
	err := d.azureIntegration.EnsureNamespace(d.cmd.Context(), d.cfg)
	if err != nil {
		return err
	}
	return d.azureIntegration.Create(d.cmd.Context(), d.cfg)
}

// NewIntegrationAzure creates the sub-command for the "integration azure"
// responsible to manage the TSSC integrations with the Azure service.
func NewIntegrationAzure(
	logger *slog.Logger,
	kube *k8s.Kube,
) *IntegrationAzure {
	azureIntegration := integrations.NewAzureIntegration(logger, kube)

	d := &IntegrationAzure{
		cmd: &cobra.Command{
			Use:          "azure [flags]",
			Short:        "Integrates a Azure instance into TSSC",
			Long:         azureIntegrationLongDesc,
			SilenceUsage: true,
		},

		logger: logger,
		kube:   kube,

		azureIntegration: azureIntegration,
	}

	azureIntegration.PersistentFlags(d.cmd)
	return d
}

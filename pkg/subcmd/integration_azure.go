package subcmd

import (
	"fmt"
	"log/slog"

	"github.com/redhat-appstudio/rhtap-cli/pkg/config"
	"github.com/redhat-appstudio/rhtap-cli/pkg/integrations"
	"github.com/redhat-appstudio/rhtap-cli/pkg/k8s"

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
Manages the Azure integration with RHTAP, by storing the required
credentials required by the RHTAP services to interact with Azure.
The credentials are stored in a Kubernetes Secret in the default
installation namespace.
`

// Cmd exposes the cobra instance.
func (d *IntegrationAzure) Cmd() *cobra.Command {
	return d.cmd
}

// Complete is a no-op in this case.
func (d *IntegrationAzure) Complete(args []string) error {
	return nil
}

// Validate checks if the required configuration is set.
func (d *IntegrationAzure) Validate() error {
	feature, err := d.cfg.GetFeature(config.RedHatDeveloperHub)
	if err != nil {
		return err
	}
	if !feature.Enabled {
		return fmt.Errorf("Red Hat Developer Hub feature is not enabled")
	}
	return d.azureIntegration.Validate()
}

// Run creates or updates the Azure integration secret.
func (d *IntegrationAzure) Run() error {
	if err := d.azureIntegration.EnsureNamespace(d.cmd.Context()); err != nil {
		return err
	}
	return d.azureIntegration.Create(d.cmd.Context())
}

// NewIntegrationAzure creates the sub-command for the "integration azure"
// responsible to manage the RHTAP integrations with the Azure service.
func NewIntegrationAzure(
	logger *slog.Logger,
	cfg *config.Config,
	kube *k8s.Kube,
) *IntegrationAzure {
	azureIntegration := integrations.NewAzureIntegration(logger, cfg, kube)

	d := &IntegrationAzure{
		cmd: &cobra.Command{
			Use:          "azure [flags]",
			Short:        "Integrates a Azure instance into RHTAP",
			Long:         azureIntegrationLongDesc,
			SilenceUsage: true,
		},

		logger: logger,
		cfg:    cfg,
		kube:   kube,

		azureIntegration: azureIntegration,
	}

	p := d.cmd.PersistentFlags()
	azureIntegration.PersistentFlags(p)
	return d
}

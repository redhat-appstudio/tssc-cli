package subcmd

import (
	"fmt"
	"log/slog"

	"github.com/redhat-appstudio/rhtap-cli/pkg/config"
	"github.com/redhat-appstudio/rhtap-cli/pkg/integrations"
	"github.com/redhat-appstudio/rhtap-cli/pkg/k8s"

	"github.com/spf13/cobra"
)

// IntegrationQuay is the sub-command for the "integration quay",
// responsible for creating and updating the Quay integration secret.
type IntegrationQuay struct {
	cmd    *cobra.Command // cobra command
	logger *slog.Logger   // application logger
	cfg    *config.Config // installer configuration
	kube   *k8s.Kube      // kubernetes client

	quayIntegration *integrations.QuayIntegration // quay integration

	apiToken         string // web API token
	dockerconfigjson string // credentials to push/pull from the registry
}

var _ Interface = &IntegrationQuay{}

const quayIntegrationLongDesc = `
Manages the Quay integration with RHTAP, by storing the required
credentials required by the RHTAP services to interact with Quay.

The credentials are stored in a Kubernetes Secret in the configured namespace
for RHDH.


The given dockerconfig must include the repository path. E.g. "quay.io" becomes "quay.io/my-repository".
The given API token (--token) must have push/pull permissions on the target repository.
`

// Cmd exposes the cobra instance.
func (d *IntegrationQuay) Cmd() *cobra.Command {
	return d.cmd
}

// Complete is a no-op in this case.
func (d *IntegrationQuay) Complete(args []string) error {
	return nil
}

// Validate checks if the required configuration is set.
func (d *IntegrationQuay) Validate() error {
	feature, err := d.cfg.GetFeature(config.RedHatDeveloperHub)
	if err != nil {
		return err
	}
	if !feature.Enabled {
		return fmt.Errorf("Red Hat Developer Hub feature is not enabled")
	}
	return d.quayIntegration.Validate()
}

// Run creates or updates the Quay integration secret.
func (d *IntegrationQuay) Run() error {
	if err := d.quayIntegration.EnsureNamespace(d.cmd.Context()); err != nil {
		return err
	}
	return d.quayIntegration.Create(d.cmd.Context())
}

// NewIntegrationQuay creates the sub-command for the "integration quay"
// responsible to manage the RHTAP integrations with a Quay image registry.
func NewIntegrationQuay(
	logger *slog.Logger,
	cfg *config.Config,
	kube *k8s.Kube,
) *IntegrationQuay {
	quayIntegration := integrations.NewQuayIntegration(logger, cfg, kube)

	d := &IntegrationQuay{
		cmd: &cobra.Command{
			Use:          "quay [flags]",
			Short:        "Integrates a Quay instance into RHTAP",
			Long:         quayIntegrationLongDesc,
			SilenceUsage: true,
		},

		logger: logger,
		cfg:    cfg,
		kube:   kube,

		quayIntegration: quayIntegration,
	}

	p := d.cmd.PersistentFlags()
	quayIntegration.PersistentFlags(p)
	return d
}

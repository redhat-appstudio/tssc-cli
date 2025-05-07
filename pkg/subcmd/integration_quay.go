package subcmd

import (
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

	apiToken                 string // web API token
	dockerconfigjson         string // credentials to push/pull from the registry
	dockerconfigjsonreadonly string // credentials to read from the registry
}

var _ Interface = &IntegrationQuay{}

const quayIntegrationLongDesc = `
Manages the Quay integration with RHTAP, by storing the required
credentials required by the RHTAP services to interact with Quay.

The credentials are stored in a Kubernetes Secret in the configured namespace
for RHDH.


If you experience push issues, add the image repository path in the "dockerconfig.json". For example, instead of "quay.io", specify the full repository path "quay.io/my-repository", as shown below:

/bin/tssc integration quay --kube-config ~/my/kube/config --dockerconfigjson '{ "auths": { "quay.io/my-repository": { "auth": "REDACTED", "email": "" }  } }' --token "REDACTED" --url 'https://quay.io'

The given API token (--token) must have push/pull permissions on the target repository.
`

// Cmd exposes the cobra instance.
func (d *IntegrationQuay) Cmd() *cobra.Command {
	return d.cmd
}

// Complete is a no-op in this case.
func (d *IntegrationQuay) Complete(args []string) error {
	var err error
	d.cfg, err = bootstrapConfig(d.cmd.Context(), d.kube)
	return err
}

// Validate checks if the required configuration is set.
func (d *IntegrationQuay) Validate() error {
	return d.quayIntegration.Validate()
}

// Run creates or updates the Quay integration secret.
func (d *IntegrationQuay) Run() error {
	err := d.quayIntegration.EnsureNamespace(d.cmd.Context(), d.cfg)
	if err != nil {
		return err
	}
	return d.quayIntegration.Create(d.cmd.Context(), d.cfg)
}

// NewIntegrationQuay creates the sub-command for the "integration quay"
// responsible to manage the RHTAP integrations with a Quay image registry.
func NewIntegrationQuay(logger *slog.Logger, kube *k8s.Kube) *IntegrationQuay {
	quayIntegration := integrations.NewQuayIntegration(logger, kube)

	d := &IntegrationQuay{
		cmd: &cobra.Command{
			Use:          "quay [flags]",
			Short:        "Integrates a Quay instance into RHTAP",
			Long:         quayIntegrationLongDesc,
			SilenceUsage: true,
		},

		logger: logger,
		kube:   kube,

		quayIntegration: quayIntegration,
	}

	p := d.cmd.PersistentFlags()
	quayIntegration.PersistentFlags(p)
	return d
}

package subcmd

import (
	"log/slog"

	"github.com/redhat-appstudio/tssc/pkg/config"
	"github.com/redhat-appstudio/tssc/pkg/integrations"
	"github.com/redhat-appstudio/tssc/pkg/k8s"

	"github.com/spf13/cobra"
)

// IntegrationJenkins is the sub-command for the "integration jenkins",
// responsible for creating and updating the Jenkins integration secret.
type IntegrationJenkins struct {
	cmd    *cobra.Command // cobra command
	logger *slog.Logger   // application logger
	cfg    *config.Config // installer configuration
	kube   *k8s.Kube      // kubernetes client

	jenkinsIntegration *integrations.JenkinsIntegration // jenkins integration

	token    string // API token
	url      string // service URL
	username string // user to connect to the service
}

var _ Interface = &IntegrationJenkins{}

const jenkinsIntegrationLongDesc = `
Manages the Jenkins integration with TSSC, by storing the required
credentials required by the TSSC services to interact with Jenkins.

The credentials are stored in a Kubernetes Secret in the configured namespace
for RHDH.
`

// Cmd exposes the cobra instance.
func (d *IntegrationJenkins) Cmd() *cobra.Command {
	return d.cmd
}

// Complete is a no-op in this case.
func (d *IntegrationJenkins) Complete(args []string) error {
	var err error
	d.cfg, err = bootstrapConfig(d.cmd.Context(), d.kube)
	return err
}

// Validate checks if the required configuration is set.
func (d *IntegrationJenkins) Validate() error {
	return d.jenkinsIntegration.Validate()
}

// Run creates or updates the Jenkins integration secret.
func (d *IntegrationJenkins) Run() error {
	err := d.jenkinsIntegration.EnsureNamespace(d.cmd.Context(), d.cfg)
	if err != nil {
		return err
	}
	return d.jenkinsIntegration.Create(d.cmd.Context(), d.cfg)
}

// NewIntegrationJenkins creates the sub-command for the "integration jenkins"
// responsible to manage the TSSC integrations with the Jenkins service.
func NewIntegrationJenkins(
	logger *slog.Logger,
	kube *k8s.Kube,
) *IntegrationJenkins {
	jenkinsIntegration := integrations.NewJenkinsIntegration(logger, kube)

	d := &IntegrationJenkins{
		cmd: &cobra.Command{
			Use:          "jenkins [flags]",
			Short:        "Integrates a Jenkins instance into TSSC",
			Long:         jenkinsIntegrationLongDesc,
			SilenceUsage: true,
		},

		logger: logger,
		kube:   kube,

		jenkinsIntegration: jenkinsIntegration,
	}

	jenkinsIntegration.PersistentFlags(d.cmd)
	return d
}

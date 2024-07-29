package subcmd

import (
	"fmt"
	"log/slog"

	"github.com/redhat-appstudio/rhtap-cli/pkg/config"
	"github.com/redhat-appstudio/rhtap-cli/pkg/integrations"
	"github.com/redhat-appstudio/rhtap-cli/pkg/k8s"

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
Manages the Jenkins integration with RHTAP, by storing the required
credentials required by the RHTAP services to interact with Jenkins.

The credentials are stored in a Kubernetes Secret in the configured namespace
for RHDH.
`

// Cmd exposes the cobra instance.
func (d *IntegrationJenkins) Cmd() *cobra.Command {
	return d.cmd
}

// Complete is a no-op in this case.
func (d *IntegrationJenkins) Complete(args []string) error {
	return nil
}

// Validate checks if the required configuration is set.
func (d *IntegrationJenkins) Validate() error {
	feature, err := d.cfg.GetFeature(config.RedHatDeveloperHub)
	if err != nil {
		return err
	}
	if !feature.Enabled {
		return fmt.Errorf("Red Hat Developer Hub feature is not enabled")
	}
	return d.jenkinsIntegration.Validate()
}

// Run creates or updates the Jenkins integration secret.
func (d *IntegrationJenkins) Run() error {
	if err := d.jenkinsIntegration.EnsureNamespace(d.cmd.Context()); err != nil {
		return err
	}
	return d.jenkinsIntegration.Create(d.cmd.Context())
}

// NewIntegrationJenkins creates the sub-command for the "integration jenkins"
// responsible to manage the RHTAP integrations with the Jenkins service.
func NewIntegrationJenkins(
	logger *slog.Logger,
	cfg *config.Config,
	kube *k8s.Kube,
) *IntegrationJenkins {
	jenkinsIntegration := integrations.NewJenkinsIntegration(logger, cfg, kube)

	d := &IntegrationJenkins{
		cmd: &cobra.Command{
			Use:          "jenkins [flags]",
			Short:        "Integrates a Jenkins instance into RHTAP",
			Long:         jenkinsIntegrationLongDesc,
			SilenceUsage: true,
		},

		logger: logger,
		cfg:    cfg,
		kube:   kube,

		jenkinsIntegration: jenkinsIntegration,
	}

	p := d.cmd.PersistentFlags()
	jenkinsIntegration.PersistentFlags(p)
	return d
}

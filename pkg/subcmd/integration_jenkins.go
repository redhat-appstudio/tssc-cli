package subcmd

import (
	"log/slog"

	"github.com/redhat-appstudio/tssc-cli/pkg/config"
	"github.com/redhat-appstudio/tssc-cli/pkg/integration"
	"github.com/redhat-appstudio/tssc-cli/pkg/k8s"

	"github.com/spf13/cobra"
)

// IntegrationJenkins is the sub-command for the "integration jenkins",
// responsible for creating and updating the Jenkins integration secret.
type IntegrationJenkins struct {
	cmd         *cobra.Command           // cobra command
	logger      *slog.Logger             // application logger
	cfg         *config.Config           // installer configuration
	kube        *k8s.Kube                // kubernetes client
	integration *integration.Integration // integration instance
}

var _ Interface = &IntegrationJenkins{}

const jenkinsIntegrationLongDesc = `
Manages the Jenkins integration with TSSC, by storing the required
credentials required by the TSSC services to interact with Jenkins.

The credentials are stored in a Kubernetes Secret in the configured namespace
for RHDH.
`

// Cmd exposes the cobra instance.
func (j *IntegrationJenkins) Cmd() *cobra.Command {
	return j.cmd
}

// Complete is a no-op in this case.
func (j *IntegrationJenkins) Complete(args []string) error {
	var err error
	j.cfg, err = bootstrapConfig(j.cmd.Context(), j.kube)
	return err
}

// Validate checks if the required configuration is set.
func (j *IntegrationJenkins) Validate() error {
	return j.integration.Validate()
}

// Run creates or updates the Jenkins integration secret.
func (j *IntegrationJenkins) Run() error {
	return j.integration.Create(j.cmd.Context(), j.cfg)
}

// NewIntegrationJenkins creates the sub-command for the "integration jenkins"
// responsible to manage the TSSC integrations with the Jenkins service.
func NewIntegrationJenkins(
	logger *slog.Logger,
	kube *k8s.Kube,
	i *integration.Integration,
) *IntegrationJenkins {
	j := &IntegrationJenkins{
		cmd: &cobra.Command{
			Use:          "jenkins [flags]",
			Short:        "Integrates a Jenkins instance into TSSC",
			Long:         jenkinsIntegrationLongDesc,
			SilenceUsage: true,
		},

		logger:      logger,
		kube:        kube,
		integration: i,
	}
	i.PersistentFlags(j.cmd)
	return j
}

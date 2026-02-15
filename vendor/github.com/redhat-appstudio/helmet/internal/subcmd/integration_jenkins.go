package subcmd

import (
	"github.com/redhat-appstudio/helmet/api"
	"github.com/redhat-appstudio/helmet/internal/config"
	"github.com/redhat-appstudio/helmet/internal/integration"
	"github.com/redhat-appstudio/helmet/internal/runcontext"

	"github.com/spf13/cobra"
)

// IntegrationJenkins is the sub-command for the "integration jenkins",
// responsible for creating and updating the Jenkins integration secret.
type IntegrationJenkins struct {
	cmd         *cobra.Command           // cobra command
	appCtx      *api.AppContext          // application context
	runCtx      *runcontext.RunContext   // run context (kube, logger, chartfs)
	cfg         *config.Config           // installer configuration
	integration *integration.Integration // integration instance
}

var _ api.SubCommand = &IntegrationJenkins{}

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
func (j *IntegrationJenkins) Complete(_ []string) error {
	var err error
	j.cfg, err = bootstrapConfig(j.cmd.Context(), j.appCtx, j.runCtx)
	return err
}

// Validate checks if the required configuration is set.
func (j *IntegrationJenkins) Validate() error {
	return j.integration.Validate()
}

// Run creates or updates the Jenkins integration secret.
func (j *IntegrationJenkins) Run() error {
	return j.integration.Create(j.cmd.Context(), j.runCtx, j.cfg)
}

// NewIntegrationJenkins creates the sub-command for the "integration jenkins"
// responsible to manage the TSSC integrations with the Jenkins service.
func NewIntegrationJenkins(
	appCtx *api.AppContext,
	runCtx *runcontext.RunContext,
	i *integration.Integration,
) *IntegrationJenkins {
	j := &IntegrationJenkins{
		cmd: &cobra.Command{
			Use:          "jenkins [flags]",
			Short:        "Integrates a Jenkins instance into TSSC",
			Long:         jenkinsIntegrationLongDesc,
			SilenceUsage: true,
		},

		appCtx:      appCtx,
		runCtx:      runCtx,
		integration: i,
	}
	i.PersistentFlags(j.cmd)
	return j
}

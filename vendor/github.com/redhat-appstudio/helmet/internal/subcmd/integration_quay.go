package subcmd

import (
	"fmt"

	"github.com/redhat-appstudio/helmet/api"
	"github.com/redhat-appstudio/helmet/internal/config"
	"github.com/redhat-appstudio/helmet/internal/integration"
	"github.com/redhat-appstudio/helmet/internal/runcontext"

	"github.com/spf13/cobra"
)

// IntegrationQuay is the sub-command for the "integration quay",
// responsible for creating and updating the Quay integration secret.
type IntegrationQuay struct {
	cmd         *cobra.Command           // cobra command
	appCtx      *api.AppContext          // application context
	runCtx      *runcontext.RunContext   // run context (kube, logger, chartfs)
	cfg         *config.Config           // installer configuration
	integration *integration.Integration // integration instance
}

var _ api.SubCommand = &IntegrationQuay{}

// Cmd exposes the cobra instance.
func (q *IntegrationQuay) Cmd() *cobra.Command {
	return q.cmd
}

// Complete is a no-op in this case.
func (q *IntegrationQuay) Complete(_ []string) error {
	var err error
	q.cfg, err = bootstrapConfig(q.cmd.Context(), q.appCtx, q.runCtx)
	return err
}

// Validate checks if the required configuration is set.
func (q *IntegrationQuay) Validate() error {
	return q.integration.Validate()
}

// Run creates or updates the Quay integration secret.
func (q *IntegrationQuay) Run() error {
	return q.integration.Create(q.cmd.Context(), q.runCtx, q.cfg)
}

// NewIntegrationQuay creates the sub-command for the "integration quay"
// responsible to manage the integrations with a Quay image registry.
func NewIntegrationQuay(
	appCtx *api.AppContext,
	runCtx *runcontext.RunContext,
	i *integration.Integration,
) *IntegrationQuay {
	q := &IntegrationQuay{
		cmd: &cobra.Command{
			Use:   "quay [flags]",
			Short: fmt.Sprintf("Integrates a Quay instance into %s", appCtx.Name),
			Long: fmt.Sprintf(`
Manages the Quay integration with %s by storing the credentials
required by %s services to interact with Quay.

The credentials are stored in a Kubernetes Secret in the namespace
configured for %s.

If you experience push issues, include the full image repository path in the
"dockerconfig.json". For example, instead of "quay.io", specify
"quay.io/my-repository", as shown below:

  $ %s integration quay \
	  --dockerconfigjson '{ "auths": { "quay.io/my-repository": { "auth": "REDACTED" } } }' \
	  --token "REDACTED" \
	  --url 'https://quay.io'

The API token (--token) must have push and pull permissions on the target
repository.`,
				appCtx.Name,
				appCtx.Name,
				appCtx.Name,
				appCtx.Name,
			),
			SilenceUsage: true,
		},

		appCtx:      appCtx,
		runCtx:      runCtx,
		integration: i,
	}
	i.PersistentFlags(q.cmd)
	return q
}

package subcmd

import (
	"log/slog"

	"github.com/redhat-appstudio/helmet/api"

	"github.com/redhat-appstudio/helmet/internal/config"
	"github.com/redhat-appstudio/helmet/internal/integration"
	"github.com/redhat-appstudio/helmet/internal/k8s"

	"github.com/spf13/cobra"
)

// IntegrationQuay is the sub-command for the "integration quay",
// responsible for creating and updating the Quay integration secret.
type IntegrationQuay struct {
	cmd         *cobra.Command           // cobra command
	appCtx      *api.AppContext          // application context
	logger      *slog.Logger             // application logger
	cfg         *config.Config           // installer configuration
	kube        *k8s.Kube                // kubernetes client
	integration *integration.Integration // integration instance
}

var _ api.SubCommand = &IntegrationQuay{}

const quayIntegrationLongDesc = `
Manages the Quay integration with TSSC, by storing the required
credentials required by the TSSC services to interact with Quay.

The credentials are stored in a Kubernetes Secret in the configured namespace
for RHDH.

If you experience push issues, add the image repository path in the
"dockerconfig.json". For example, instead of "quay.io", specify the full
repository path "quay.io/my-repository", as shown below:

  $ tssc integration quay \
	  --dockerconfigjson '{ "auths": { "quay.io/my-repository": { "auth": "REDACTED" } } }' \
	  --token "REDACTED" \
	  --url 'https://quay.io'

The given API token (--token) must have push/pull permissions on the target
repository.
`

// Cmd exposes the cobra instance.
func (q *IntegrationQuay) Cmd() *cobra.Command {
	return q.cmd
}

// Complete is a no-op in this case.
func (q *IntegrationQuay) Complete(args []string) error {
	var err error
	q.cfg, err = bootstrapConfig(q.cmd.Context(), q.appCtx, q.kube)
	return err
}

// Validate checks if the required configuration is set.
func (q *IntegrationQuay) Validate() error {
	return q.integration.Validate()
}

// Run creates or updates the Quay integration secret.
func (q *IntegrationQuay) Run() error {
	return q.integration.Create(q.cmd.Context(), q.cfg)
}

// NewIntegrationQuay creates the sub-command for the "integration quay"
// responsible to manage the TSSC integrations with a Quay image registry.
func NewIntegrationQuay(
	appCtx *api.AppContext,
	logger *slog.Logger,
	kube *k8s.Kube,
	i *integration.Integration,
) *IntegrationQuay {
	q := &IntegrationQuay{
		cmd: &cobra.Command{
			Use:          "quay [flags]",
			Short:        "Integrates a Quay instance into TSSC",
			Long:         quayIntegrationLongDesc,
			SilenceUsage: true,
		},

		appCtx:      appCtx,
		logger:      logger,
		kube:        kube,
		integration: i,
	}
	i.PersistentFlags(q.cmd)
	return q
}

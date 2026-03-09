package subcmd

import (
	"fmt"

	"github.com/redhat-appstudio/helmet/api"
	"github.com/redhat-appstudio/helmet/internal/config"
	"github.com/redhat-appstudio/helmet/internal/integration"
	"github.com/redhat-appstudio/helmet/internal/runcontext"

	"github.com/spf13/cobra"
)

// IntegrationTrustificationAuth is the sub-command for the "integration
// trustificationauth", responsible for creating and updating
// the trustificationauth integration secret.
type IntegrationTrustificationAuth struct {
	cmd         *cobra.Command           // cobra command
	appCtx      *api.AppContext          // application context
	runCtx      *runcontext.RunContext   // run context (kube, logger, chartfs)
	cfg         *config.Config           // installer configuration
	integration *integration.Integration // integration instance
}

var _ api.SubCommand = &IntegrationTrustificationAuth{}

// Cmd exposes the cobra instance.
func (t *IntegrationTrustificationAuth) Cmd() *cobra.Command {
	return t.cmd
}

// Complete is a no-op in this case.
func (t *IntegrationTrustificationAuth) Complete(_ []string) error {
	var err error
	t.cfg, err = bootstrapConfig(t.cmd.Context(), t.appCtx, t.runCtx)
	return err
}

// Validate checks if the required configuration is set.
func (t *IntegrationTrustificationAuth) Validate() error {
	return t.integration.Validate()
}

// Run creates or updates the trustificationauth integration secret.
func (t *IntegrationTrustificationAuth) Run() error {
	return t.integration.Create(t.cmd.Context(), t.runCtx, t.cfg)
}

// NewIntegrationTrustificationAuth creates the sub-command for the "integration
// trustificationauth" responsible to manage the authentication integrations with the
// Trustification service.
func NewIntegrationTrustificationAuth(
	appCtx *api.AppContext,
	runCtx *runcontext.RunContext,
	i *integration.Integration,
) *IntegrationTrustificationAuth {
	t := &IntegrationTrustificationAuth{
		cmd: &cobra.Command{
			Aliases: []string{"trustification-auth"},
			Use:     "trustification-auth [flags]",
			Short: fmt.Sprintf(
				"Integrates a trustification-auth instance into %s",
				appCtx.Name,
			),
			Long: fmt.Sprintf(`
Manages the trustification-auth integration with %s by storing the credentials
required by %s services to interact with Trustification.

The credentials are stored in a Kubernetes Secret in the namespace
configured for %s.`,
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
	i.PersistentFlags(t.cmd)
	return t
}

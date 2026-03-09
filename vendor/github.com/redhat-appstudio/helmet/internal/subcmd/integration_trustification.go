package subcmd

import (
	"fmt"

	"github.com/redhat-appstudio/helmet/api"
	"github.com/redhat-appstudio/helmet/internal/config"
	"github.com/redhat-appstudio/helmet/internal/integration"
	"github.com/redhat-appstudio/helmet/internal/runcontext"

	"github.com/spf13/cobra"
)

// IntegrationTrustification is the sub-command for the "integration trustification",
// responsible for creating and updating the Trustification integration secret.
// It includes the coordinates to connect the cluster with remote Trustification services.
type IntegrationTrustification struct {
	cmd         *cobra.Command           // cobra command
	appCtx      *api.AppContext          // application context
	runCtx      *runcontext.RunContext   // run context (kube, logger, chartfs)
	cfg         *config.Config           // installer configuration
	integration *integration.Integration // integration instance
}

var _ api.SubCommand = &IntegrationTrustification{}

// Cmd exposes the cobra instance.
func (t *IntegrationTrustification) Cmd() *cobra.Command {
	return t.cmd
}

// Complete is a no-op in this case.
func (t *IntegrationTrustification) Complete(_ []string) error {
	var err error
	t.cfg, err = bootstrapConfig(t.cmd.Context(), t.appCtx, t.runCtx)
	return err
}

// Validate checks if the required configuration is set.
func (t *IntegrationTrustification) Validate() error {
	return t.integration.Validate()
}

// Run creates or updates the Trustification integration secret.
func (t *IntegrationTrustification) Run() error {
	return t.integration.Create(t.cmd.Context(), t.runCtx, t.cfg)
}

// NewIntegrationTrustification creates the sub-command for the "integration
// trustification" responsible to manage the integrations with the
// Trustification service.
func NewIntegrationTrustification(
	appCtx *api.AppContext,
	runCtx *runcontext.RunContext,
	i *integration.Integration,
) *IntegrationTrustification {
	t := &IntegrationTrustification{
		cmd: &cobra.Command{
			Use: "trustification [flags]",
			Short: fmt.Sprintf(
				"Integrates a Trustification instance into %s",
				appCtx.Name,
			),
			Long: fmt.Sprintf(`
Manages the Trustification integration with %s by storing the information
required by %s services to interact with Trustification.

The information is stored in a Kubernetes Secret in the namespace
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

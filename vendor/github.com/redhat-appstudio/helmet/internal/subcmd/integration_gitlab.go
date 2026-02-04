package subcmd

import (
	"log/slog"

	"github.com/redhat-appstudio/helmet/api"

	"github.com/redhat-appstudio/helmet/internal/config"
	"github.com/redhat-appstudio/helmet/internal/integration"
	"github.com/redhat-appstudio/helmet/internal/k8s"

	"github.com/spf13/cobra"
)

// IntegrationGitLab is the sub-command for the "integration gitlab",
// responsible for creating and updating the GitLab integration secret.
type IntegrationGitLab struct {
	cmd         *cobra.Command           // cobra command
	appCtx      *api.AppContext          // application context
	logger      *slog.Logger             // application logger
	cfg         *config.Config           // installer configuration
	kube        *k8s.Kube                // kubernetes client
	integration *integration.Integration // integration instance
}

var _ api.SubCommand = &IntegrationGitLab{}

const gitlabIntegrationLongDesc = `
Manages the GitLab integration with TSSC, by storing the required
credentials required by the TSSC services to interact with GitLab.

The credentials are stored in a Kubernetes Secret in the configured namespace
for RHDH.
`

// Cmd exposes the cobra instance.
func (g *IntegrationGitLab) Cmd() *cobra.Command {
	return g.cmd
}

// Complete is a no-op in this case.
func (g *IntegrationGitLab) Complete(args []string) error {
	var err error
	g.cfg, err = bootstrapConfig(g.cmd.Context(), g.appCtx, g.kube)
	return err
}

// Validate checks if the required configuration is set.
func (g *IntegrationGitLab) Validate() error {
	return g.integration.Validate()
}

// Run creates or updates the GitLab integration secret.
func (g *IntegrationGitLab) Run() error {
	return g.integration.Create(g.cmd.Context(), g.cfg)
}

// NewIntegrationGitLab creates the sub-command for the "integration gitlab"
// responsible to manage the TSSC integrations with the GitLab service.
func NewIntegrationGitLab(
	appCtx *api.AppContext,
	logger *slog.Logger,
	kube *k8s.Kube,
	i *integration.Integration,
) *IntegrationGitLab {
	g := &IntegrationGitLab{
		cmd: &cobra.Command{
			Use:          "gitlab [flags]",
			Short:        "Integrates a GitLab instance into TSSC",
			Long:         gitlabIntegrationLongDesc,
			SilenceUsage: true,
		},

		appCtx:      appCtx,
		logger:      logger,
		kube:        kube,
		integration: i,
	}
	i.PersistentFlags(g.cmd)
	return g
}

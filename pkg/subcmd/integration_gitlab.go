package subcmd

import (
	"log/slog"

	"github.com/redhat-appstudio/tssc/pkg/config"
	"github.com/redhat-appstudio/tssc/pkg/integrations"
	"github.com/redhat-appstudio/tssc/pkg/k8s"

	"github.com/spf13/cobra"
)

// IntegrationGitLab is the sub-command for the "integration gitlab",
// responsible for creating and updating the GitLab integration secret.
type IntegrationGitLab struct {
	cmd    *cobra.Command // cobra command
	logger *slog.Logger   // application logger
	cfg    *config.Config // installer configuration
	kube   *k8s.Kube      // kubernetes client

	gitlabIntegration *integrations.GitLabIntegration // gitlab integration
}

var _ Interface = &IntegrationGitLab{}

const gitlabIntegrationLongDesc = `
Manages the GitLab integration with TSSC, by storing the required
credentials required by the TSSC services to interact with GitLab.

The credentials are stored in a Kubernetes Secret in the configured namespace
for RHDH.
`

// Cmd exposes the cobra instance.
func (d *IntegrationGitLab) Cmd() *cobra.Command {
	return d.cmd
}

// Complete is a no-op in this case.
func (d *IntegrationGitLab) Complete(args []string) error {
	var err error
	d.cfg, err = bootstrapConfig(d.cmd.Context(), d.kube)
	return err
}

// Validate checks if the required configuration is set.
func (d *IntegrationGitLab) Validate() error {
	return d.gitlabIntegration.Validate()
}

// Run creates or updates the GitLab integration secret.
func (d *IntegrationGitLab) Run() error {
	err := d.gitlabIntegration.EnsureNamespace(d.cmd.Context(), d.cfg)
	if err != nil {
		return err
	}
	return d.gitlabIntegration.Create(d.cmd.Context(), d.cfg)
}

// NewIntegrationGitLab creates the sub-command for the "integration gitlab"
// responsible to manage the TSSC integrations with the GitLab service.
func NewIntegrationGitLab(
	logger *slog.Logger,
	kube *k8s.Kube,
) *IntegrationGitLab {
	gitlabIntegration := integrations.NewGitLabIntegration(logger, kube)

	d := &IntegrationGitLab{
		cmd: &cobra.Command{
			Use:          "gitlab [flags]",
			Short:        "Integrates a GitLab instance into TSSC",
			Long:         gitlabIntegrationLongDesc,
			SilenceUsage: true,
		},

		logger: logger,
		kube:   kube,

		gitlabIntegration: gitlabIntegration,
	}

	p := d.cmd.PersistentFlags()
	gitlabIntegration.PersistentFlags(p)
	return d
}

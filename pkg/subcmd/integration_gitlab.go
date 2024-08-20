package subcmd

import (
	"fmt"
	"log/slog"

	"github.com/redhat-appstudio/rhtap-cli/pkg/config"
	"github.com/redhat-appstudio/rhtap-cli/pkg/integrations"
	"github.com/redhat-appstudio/rhtap-cli/pkg/k8s"

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

	host  string // E.g. 'gitlab.com'
	token string // API token
}

var _ Interface = &IntegrationGitLab{}

const gitlabIntegrationLongDesc = `
Manages the GitLab integration with RHTAP, by storing the required
credentials required by the RHTAP services to interact with GitLab.

The credentials are stored in a Kubernetes Secret in the configured namespace
for RHDH.
`

// Cmd exposes the cobra instance.
func (d *IntegrationGitLab) Cmd() *cobra.Command {
	return d.cmd
}

// Complete is a no-op in this case.
func (d *IntegrationGitLab) Complete(args []string) error {
	return nil
}

// Validate checks if the required configuration is set.
func (d *IntegrationGitLab) Validate() error {
	feature, err := d.cfg.GetFeature(config.RedHatDeveloperHub)
	if err != nil {
		return err
	}
	if !feature.Enabled {
		return fmt.Errorf("Red Hat Developer Hub feature is not enabled")
	}
	return d.gitlabIntegration.Validate()
}

// Run creates or updates the GitLab integration secret.
func (d *IntegrationGitLab) Run() error {
	if err := d.gitlabIntegration.EnsureNamespace(d.cmd.Context()); err != nil {
		return err
	}
	return d.gitlabIntegration.Create(d.cmd.Context())
}

// NewIntegrationGitLab creates the sub-command for the "integration gitlab"
// responsible to manage the RHTAP integrations with the GitLab service.
func NewIntegrationGitLab(
	logger *slog.Logger,
	cfg *config.Config,
	kube *k8s.Kube,
) *IntegrationGitLab {
	gitlabIntegration := integrations.NewGitLabIntegration(logger, cfg, kube)

	d := &IntegrationGitLab{
		cmd: &cobra.Command{
			Use:          "gitlab [flags]",
			Short:        "Integrates a GitLab instance into RHTAP",
			Long:         gitlabIntegrationLongDesc,
			SilenceUsage: true,
		},

		logger: logger,
		cfg:    cfg,
		kube:   kube,

		gitlabIntegration: gitlabIntegration,
	}

	p := d.cmd.PersistentFlags()
	gitlabIntegration.PersistentFlags(p)
	return d
}

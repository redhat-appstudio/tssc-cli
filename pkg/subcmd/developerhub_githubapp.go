package subcmd

import (
	"fmt"
	"log/slog"

	"github.com/redhat-appstudio/rhtap-cli/pkg/config"
	"github.com/redhat-appstudio/rhtap-cli/pkg/constants"
	"github.com/redhat-appstudio/rhtap-cli/pkg/githubapp"
	"github.com/redhat-appstudio/rhtap-cli/pkg/k8s"
	"github.com/redhat-appstudio/rhtap-cli/pkg/rhdh"

	"github.com/spf13/cobra"
)

// DeveloperHubGitHubApp is the sub-command for the "developer-hub github-app",
// responsible for creating and updating GitHub Apps and link its configuration
// for the Developer Hub integration.
type DeveloperHubGitHubApp struct {
	cmd    *cobra.Command // cobra command
	logger *slog.Logger   // application logger
	cfg    *config.Config // installer configuration
	kube   *k8s.Kube      // kubernetes client

	gitHubIntegration *rhdh.GithubIntegration // rhdh github integration

	name   string // github app name
	create bool   // create a new github app
	update bool   // update a existing github app
}

var _ Interface = &DeveloperHubGitHubApp{}

const developerHubLongDesc = `
Manages the GitHub App integration with the Developer Hub, by creating a new
application using the GitHub API, storing the credentials required by RHDH and
others services to interact with the GitHub App.

The App credentials are stored on a Kubernetes Secret on the configured namespace
for RHDH.

The informed personal access token (--token) must have the desired permissions for
OpenShift GitOps interact with the repositores, adding "push" permission may be
required.
`

// Cmd exposes the cobra instance.
func (d *DeveloperHubGitHubApp) Cmd() *cobra.Command {
	return d.cmd
}

// Complete captures the application name, and ensures it's ready to run.
func (d *DeveloperHubGitHubApp) Complete(args []string) error {
	if d.create && d.update {
		return fmt.Errorf("cannot create and update at the same time")
	}
	if !d.create && !d.update {
		return fmt.Errorf("either create or update must be set")
	}

	if len(args) != 1 {
		return fmt.Errorf("expected 1, got %d arguments", len(args))
	}
	d.name = args[0]
	return nil
}

// Validate checks if the required configuration is set.
func (d *DeveloperHubGitHubApp) Validate() error {
	feature, err := d.cfg.GetFeature(config.RedHatDeveloperHub)
	if err != nil {
		return err
	}
	if !feature.Enabled {
		return fmt.Errorf("Red Hat Developer Hub feature is not enabled")
	}
	feature, err = d.cfg.GetFeature(config.OpenShiftPipelines)
	if err != nil {
		return err
	}
	if !feature.Enabled {
		return fmt.Errorf("OpenShift Pipelines feature is not enabled")
	}
	// TODO: make the name optional, the user will inform the GitHub App name on
	// the web-form, which can be later extracted.
	if d.name == "" {
		return fmt.Errorf("GitHub App name is required")
	}
	// Making sure the GitHub integration is valid, for instance, the required
	// personal access token is informed.
	return d.gitHubIntegration.Validate()
}

// Run creates or updates the Developer Hub GitHub App integration.
func (d *DeveloperHubGitHubApp) Run() error {
	if err := d.gitHubIntegration.EnsureNamespace(d.cmd.Context()); err != nil {
		return err
	}
	if d.create {
		return d.gitHubIntegration.Create(d.cmd.Context(), d.name)
	}
	if d.update {
		// TODO: implement update.
		panic(fmt.Sprintf(
			"TODO: '%s developer-hub github-app --update'", constants.AppName,
		))
	}
	return nil
}

// NewDeveloperHubGitHubApp creates the sub-command for the "developer-hub
// github-app", responsible to manage the DH integrations with a GitHub App.
func NewDeveloperHubGitHubApp(
	logger *slog.Logger,
	cfg *config.Config,
	kube *k8s.Kube,
) *DeveloperHubGitHubApp {
	gitHubApp := githubapp.NewGitHubApp(logger)
	gitHubIntegration := rhdh.NewGithubIntegration(logger, cfg, kube, gitHubApp)

	d := &DeveloperHubGitHubApp{
		cmd: &cobra.Command{
			Use:          "github-app <name> [--create|--update] [flags]",
			Short:        "Prepares a GitHub App for DeveloperHub integration",
			Long:         developerHubLongDesc,
			SilenceUsage: true,
		},

		logger: logger,
		cfg:    cfg,
		kube:   kube,

		gitHubIntegration: gitHubIntegration,

		create: false,
		update: false,
	}

	p := d.cmd.PersistentFlags()
	p.BoolVar(&d.create, "create", d.create, "Create a new GitHub App")
	p.BoolVar(&d.update, "update", d.update, "Update an existing GitHub App")
	gitHubIntegration.PersistentFlags(p)
	gitHubApp.PersistentFlags(p)
	return d
}

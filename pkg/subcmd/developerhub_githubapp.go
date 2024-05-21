package subcmd

import (
	"fmt"
	"log/slog"

	"github.com/redhat-appstudio/rhtap-cli/pkg/config"
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

	return k8s.EnsureOpenShiftProject(
		d.cmd.Context(),
		d.logger,
		d.kube,
		d.cfg.Installer.Namespace,
	)
}

// Validate checks if the required configuration is set.
func (d *DeveloperHubGitHubApp) Validate() error {
	if !d.cfg.Installer.Features.RedHatDeveloperHub.Enabled {
		return fmt.Errorf("Red Hat Developer Hub feature is not enabled")
	}
	if !d.cfg.Installer.Features.OpenShiftPipelines.Enabled {
		return fmt.Errorf("OpenShift Pipelines feature is not enabled")
	}
	if d.name == "" {
		return fmt.Errorf("GitHub App name is required")
	}
	return nil
}

// Run creates or updates the Developer Hub GitHub App integration.
func (d *DeveloperHubGitHubApp) Run() error {
	if d.create {
		return d.gitHubIntegration.Create(d.cmd.Context(), d.name)
	}
	if d.update {
		// TODO: implement update.
		panic("TODO: 'rhtap-installer-cli developer-hub github-app --update'")
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

package subcmd

import (
	"fmt"
	"log/slog"

	"github.com/redhat-appstudio/helmet/api"

	"github.com/redhat-appstudio/helmet/internal/config"
	"github.com/redhat-appstudio/helmet/internal/integration"
	"github.com/redhat-appstudio/helmet/internal/k8s"

	"github.com/spf13/cobra"
)

// IntegrationGitHub is the sub-command for the "integration github-app",
// responsible for creating and updating the GitHub Apps integration secret.
type IntegrationGitHub struct {
	cmd         *cobra.Command           // cobra command
	appCtx      *api.AppContext          // application context
	logger      *slog.Logger             // application logger
	cfg         *config.Config           // installer configuration
	kube        *k8s.Kube                // kubernetes client
	integration *integration.Integration // integration instance

	create bool // create a new github app
	update bool // update a existing github app
}

var _ api.SubCommand = &IntegrationGitHub{}

const integrationLongDesc = `
Manages the GitHub App integration with TSSC, by creating a new application
using the GitHub API, and storing the credentials required by the TSSC services
to interact with the GitHub App.

The App credentials are stored in a Kubernetes Secret in the configured namespace
for RHDH.

The given personal access token (--token) must have the desired permissions for
OpenShift GitOps and Openshift Pipelines to interact with the repositores, adding
"push" permission may be required.
`

// Cmd exposes the cobra instance.
func (g *IntegrationGitHub) Cmd() *cobra.Command {
	return g.cmd
}

// Complete captures the application name, and ensures it's ready to run.
func (g *IntegrationGitHub) Complete(args []string) error {
	var err error
	g.cfg, err = bootstrapConfig(g.cmd.Context(), g.appCtx, g.kube)
	if err != nil {
		return err
	}

	if g.create && g.update {
		return fmt.Errorf("cannot create and update at the same time")
	}
	if !g.create && !g.update {
		return fmt.Errorf("either create or update must be set")
	}

	if len(args) != 1 {
		return fmt.Errorf(
			"expected 1, got %d arguments. The GitHub App name is required.",
			len(args),
		)
	}
	return g.integration.SetArgument(integration.GitHubAppName, args[0])
}

// Validate checks if the required configuration is set.
func (g *IntegrationGitHub) Validate() error {
	return g.integration.Validate()
}

// Manages the GitHub App and integration secret.
func (g *IntegrationGitHub) Run() error {
	if g.create {
		return g.integration.Create(g.cmd.Context(), g.cfg)
	}
	if g.update {
		// TODO: implement update.
		panic(fmt.Sprintf(
			"TODO: '%s integration github --update'", g.appCtx.Name,
		))
	}
	return nil
}

// NewIntegrationGitHub creates the sub-command for the "integration github",
// which manages the TSSC integration with a GitHub App.
func NewIntegrationGitHub(
	appCtx *api.AppContext,
	logger *slog.Logger,
	kube *k8s.Kube,
	i *integration.Integration,
) *IntegrationGitHub {
	g := &IntegrationGitHub{
		cmd: &cobra.Command{
			Aliases:      []string{"github-app"},
			Use:          "github <name> [--create|--update] [flags]",
			Short:        "Prepares a GitHub App for TSSC integration",
			Long:         integrationLongDesc,
			SilenceUsage: true,
		},

		appCtx:      appCtx,
		logger:      logger,
		kube:        kube,
		integration: i,

		create: false,
		update: false,
	}
	p := g.cmd.PersistentFlags()
	p.BoolVar(&g.create, "create", g.create, "Create a new GitHub App")
	p.BoolVar(&g.update, "update", g.update, "Update an existing GitHub App")
	i.PersistentFlags(g.cmd)
	return g
}

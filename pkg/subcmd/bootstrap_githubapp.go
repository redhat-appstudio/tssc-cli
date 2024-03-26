package subcmd

import (
	"fmt"
	"log/slog"

	"github.com/otaviof/rhtap-installer-cli/pkg/bootstrap"
	"github.com/otaviof/rhtap-installer-cli/pkg/config"
	"github.com/otaviof/rhtap-installer-cli/pkg/k8s"
	"github.com/spf13/cobra"
	"k8s.io/apimachinery/pkg/types"
)

type BootstrapGitHubApp struct {
	cmd       *cobra.Command       // cobra command
	logger    *slog.Logger         // application logger
	cfg       *config.Spec         // installer configuration
	kube      *k8s.Kube            // kubernetes client
	gitHubApp *bootstrap.GitHubApp // GitHubApp instance

	update bool // update flag
}

var _ Interface = &BootstrapGitHubApp{}

// ErrAppNameNotInformed is returned when app name is empty.
var ErrAppNameNotInformed = fmt.Errorf("app name not informed")

func (b *BootstrapGitHubApp) Cmd() *cobra.Command {
	return b.cmd
}

func (b *BootstrapGitHubApp) Complete(args []string) error {
	if len(args) != 1 {
		return fmt.Errorf("%w: expected 1, got %d",
			ErrAppNameNotInformed, len(args))
	}
	b.gitHubApp.SetName(args[0])
	return nil
}

func (b *BootstrapGitHubApp) Validate() error {
	return nil
}

func (b *BootstrapGitHubApp) Run() error {
	ctx := b.cmd.Context()
	if b.update {
		return b.gitHubApp.Update()
	}
	appConfig, err := b.gitHubApp.Create(ctx)
	if err != nil {
		return err
	}
	secretName := types.NamespacedName{}
	return bootstrap.CreateGitHubAppConfigSecret(
		ctx, b.kube, secretName, appConfig)
}

func NewBootstrapGitHubApp(
	logger *slog.Logger,
	cfg *config.Spec,
	kube *k8s.Kube,
) *BootstrapGitHubApp {
	b := &BootstrapGitHubApp{
		cmd: &cobra.Command{
			Use:   "github-app <name> [flags]",
			Short: "Bootstrap a GitHub App for RHTAP",
		},
		logger:    logger,
		cfg:       cfg,
		kube:      kube,
		gitHubApp: bootstrap.NewGitHubApp(logger),
	}
	p := b.cmd.PersistentFlags()
	p.BoolVar(&b.update, "update", false, "update the GitHub App")
	b.gitHubApp.PersistentFlags(p)
	return b
}

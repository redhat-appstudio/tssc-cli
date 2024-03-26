package subcmd

import (
	"log/slog"

	"github.com/otaviof/rhtap-installer-cli/pkg/config"
	"github.com/otaviof/rhtap-installer-cli/pkg/k8s"
	"github.com/spf13/cobra"
)

func NewBootstrap(
	logger *slog.Logger,
	cfg *config.Spec,
	kube *k8s.Kube,
) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "bootstrap <type>",
		Short: "Bootstraps the VCS provider for RHTAP",
	}
	runner := NewRunner(NewBootstrapGitHubApp(logger, cfg, kube))
	cmd.AddCommand(runner.Cmd())
	return cmd
}

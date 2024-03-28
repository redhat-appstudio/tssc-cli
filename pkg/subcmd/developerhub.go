package subcmd

import (
	"log/slog"

	"github.com/otaviof/rhtap-installer-cli/pkg/config"
	"github.com/otaviof/rhtap-installer-cli/pkg/k8s"
	"github.com/spf13/cobra"
)

func NewDeveloperHub(
	logger *slog.Logger,
	cfg *config.Config,
	kube *k8s.Kube,
) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "developer-hub <type>",
		Short: "Configures the VCS provider for the DeveloperHub",
	}

	cmd.AddCommand(NewRunner(NewDeveloperHubGitHubApp(logger, cfg, kube)).Cmd())
	return cmd
}

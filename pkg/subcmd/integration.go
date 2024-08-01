package subcmd

import (
	"log/slog"

	"github.com/redhat-appstudio/rhtap-cli/pkg/config"
	"github.com/redhat-appstudio/rhtap-cli/pkg/k8s"
	"github.com/spf13/cobra"
)

func NewIntegration(
	logger *slog.Logger,
	cfg *config.Config,
	kube *k8s.Kube,
) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "integration <type>",
		Short: "Configures an external service provider for RHTAP",
	}

	cmd.AddCommand(NewRunner(NewIntegrationACS(logger, cfg, kube)).Cmd())
	cmd.AddCommand(NewRunner(NewIntegrationGitHubApp(logger, cfg, kube)).Cmd())
	cmd.AddCommand(NewRunner(NewIntegrationGitLab(logger, cfg, kube)).Cmd())
	cmd.AddCommand(NewRunner(NewIntegrationJenkins(logger, cfg, kube)).Cmd())
	cmd.AddCommand(NewRunner(NewIntegrationQuay(logger, cfg, kube)).Cmd())
	cmd.AddCommand(NewRunner(NewIntegrationTrustification(logger, cfg, kube)).Cmd())
	return cmd
}

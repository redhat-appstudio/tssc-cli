package subcmd

import (
	"log/slog"

	"github.com/redhat-appstudio/tssc-cli/pkg/k8s"

	"github.com/spf13/cobra"
)

func NewIntegration(logger *slog.Logger, kube *k8s.Kube) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "integration <type>",
		Short: "Configures an external service provider for TSSC",
	}

	cmd.AddCommand(NewRunner(NewIntegrationACS(logger, kube)).Cmd())
	cmd.AddCommand(NewRunner(NewIntegrationArtifactory(logger, kube)).Cmd())
	cmd.AddCommand(NewRunner(NewIntegrationAzure(logger, kube)).Cmd())
	cmd.AddCommand(NewRunner(NewIntegrationBitBucket(logger, kube)).Cmd())
	cmd.AddCommand(NewRunner(NewIntegrationGitHubApp(logger, kube)).Cmd())
	cmd.AddCommand(NewRunner(NewIntegrationGitLab(logger, kube)).Cmd())
	cmd.AddCommand(NewRunner(NewIntegrationJenkins(logger, kube)).Cmd())
	cmd.AddCommand(NewRunner(NewIntegrationNexus(logger, kube)).Cmd())
	cmd.AddCommand(NewRunner(NewIntegrationQuay(logger, kube)).Cmd())
	cmd.AddCommand(NewRunner(NewIntegrationTrustification(logger, kube)).Cmd())

	return cmd
}

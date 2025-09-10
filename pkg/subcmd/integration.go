package subcmd

import (
	"log/slog"

	"github.com/redhat-appstudio/tssc-cli/pkg/integrations"
	"github.com/redhat-appstudio/tssc-cli/pkg/k8s"

	"github.com/spf13/cobra"
)

func NewIntegration(logger *slog.Logger, kube *k8s.Kube) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "integration <type>",
		Short: "Configures an external service provider for TSSC",
	}

	manager := integrations.NewManager(logger, kube)

	for _, integration := range []Interface{
		NewIntegrationACS(
			logger, kube, manager.Get(integrations.ACS)),
		NewIntegrationArtifactory(
			logger, kube, manager.Get(integrations.Artifactory)),
		NewIntegrationAzure(
			logger, kube, manager.Get(integrations.Azure)),
		NewIntegrationBitBucket(
			logger, kube, manager.Get(integrations.BitBucket)),
		NewIntegrationGitHubApp(
			logger, kube, manager.Get(integrations.GitHubApp)),
		NewIntegrationGitLab(
			logger, kube, manager.Get(integrations.GitLab)),
		NewIntegrationJenkins(
			logger, kube, manager.Get(integrations.Jenkins)),
		NewIntegrationNexus(
			logger, kube, manager.Get(integrations.Nexus)),
		NewIntegrationQuay(
			logger, kube, manager.Get(integrations.Quay)),
		NewIntegrationTrustification(
			logger, kube, manager.Get(integrations.Trustification)),
	} {
		cmd.AddCommand(NewRunner(integration).Cmd())
	}

	return cmd
}

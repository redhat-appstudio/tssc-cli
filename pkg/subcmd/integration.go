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
			logger, kube, manager.Integration(integrations.ACS)),
		NewIntegrationArtifactory(
			logger, kube, manager.Integration(integrations.Artifactory)),
		NewIntegrationAzure(
			logger, kube, manager.Integration(integrations.Azure)),
		NewIntegrationBitBucket(
			logger, kube, manager.Integration(integrations.BitBucket)),
		NewIntegrationGitHub(
			logger, kube, manager.Integration(integrations.GitHub)),
		NewIntegrationGitLab(
			logger, kube, manager.Integration(integrations.GitLab)),
		NewIntegrationJenkins(
			logger, kube, manager.Integration(integrations.Jenkins)),
		NewIntegrationNexus(
			logger, kube, manager.Integration(integrations.Nexus)),
		NewIntegrationQuay(
			logger, kube, manager.Integration(integrations.Quay)),
		NewIntegrationTrustedArtifactSigner(
			logger, kube, manager.Integration(integrations.TrustedArtifactSigner)),
		NewIntegrationTrustification(
			logger, kube, manager.Integration(integrations.Trustification)),
	} {
		cmd.AddCommand(NewRunner(integration).Cmd())
	}

	return cmd
}

package subcmd

import (
	"log/slog"

	"github.com/redhat-appstudio/tssc-cli/pkg/api"
	"github.com/redhat-appstudio/tssc-cli/pkg/chartfs"
	"github.com/redhat-appstudio/tssc-cli/pkg/config"
	"github.com/redhat-appstudio/tssc-cli/pkg/integrations"
	"github.com/redhat-appstudio/tssc-cli/pkg/k8s"
	"github.com/redhat-appstudio/tssc-cli/pkg/resolver"

	"github.com/spf13/cobra"
)

func NewIntegration(
	logger *slog.Logger,
	kube *k8s.Kube,
	cfs *chartfs.ChartFS,
) *cobra.Command {
	manager := integrations.NewManager(logger, kube)

	cmd := &cobra.Command{
		Use:   "integration <type>",
		Short: "Configures an external service provider for TSSC",
		PersistentPostRunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := bootstrapConfig(cmd.Context(), kube)
			if err != nil {
				return err
			}

			charts, err := cfs.GetAllCharts()
			if err != nil {
				return err
			}

			collection, err := resolver.NewCollection(charts)
			if err != nil {
				return err
			}

			configuredIntegrations, err := manager.ConfiguredIntegrations(cmd.Context(), cfg)
			if err != nil {
				return err
			}

			updated := false
			for _, integrationName := range configuredIntegrations {
				productName := collection.GetProductNameForIntegration(integrationName)
				if productName == "" {
					continue
				}

				spec, err := cfg.GetProduct(productName)
				if err != nil {
					return err
				}

				if spec.Enabled {
					spec.Enabled = false
					if err := cfg.SetProduct(productName, *spec); err != nil {
						return err
					}
					updated = true
				}
			}

			if updated {
				return config.NewConfigMapManager(kube).Update(cmd.Context(), cfg)
			}

			return nil
		},
	}

	for _, integration := range []api.SubCommand{
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
		cmd.AddCommand(api.NewRunner(integration).Cmd())
	}

	return cmd
}

package subcmd

import (
	"log/slog"

	"github.com/redhat-appstudio/helmet/api"
	"github.com/redhat-appstudio/helmet/internal/chartfs"
	"github.com/redhat-appstudio/helmet/internal/config"
	"github.com/redhat-appstudio/helmet/internal/integrations"
	"github.com/redhat-appstudio/helmet/internal/k8s"
	"github.com/redhat-appstudio/helmet/internal/resolver"

	"github.com/spf13/cobra"
)

func NewIntegration(
	appCtx *api.AppContext,
	logger *slog.Logger,
	kube *k8s.Kube,
	cfs *chartfs.ChartFS,
	manager *integrations.Manager,
) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "integration <type>",
		Short: "Configures an external service provider for TSSC",
		PersistentPostRunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := bootstrapConfig(cmd.Context(), appCtx, kube)
			if err != nil {
				return err
			}

			charts, err := cfs.GetAllCharts()
			if err != nil {
				return err
			}

			collection, err := resolver.NewCollection(appCtx, charts)
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
				return config.NewConfigMapManager(kube, appCtx.Name).
					Update(cmd.Context(), cfg)
			}

			return nil
		},
	}

	for _, mod := range manager.GetModules() {
		wrapper := manager.Integration(integrations.IntegrationName(mod.Name))
		sub := mod.Command(appCtx, logger, kube, wrapper)
		cmd.AddCommand(api.NewRunner(sub).Cmd())
	}

	return cmd
}

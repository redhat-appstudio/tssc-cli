package subcmd

import (
	"github.com/redhat-appstudio/helmet/api"
	"github.com/redhat-appstudio/helmet/internal/config"
	"github.com/redhat-appstudio/helmet/internal/integrations"
	"github.com/redhat-appstudio/helmet/internal/resolver"
	"github.com/redhat-appstudio/helmet/internal/runcontext"

	"github.com/spf13/cobra"
)

func NewIntegration(
	appCtx *api.AppContext,
	runCtx *runcontext.RunContext,
	manager *integrations.Manager,
) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "integration <type>",
		Short: "Configures an external service provider for TSSC",
		PersistentPostRunE: func(cmd *cobra.Command, _ []string) error {
			ctx := cmd.Context()
			cfg, err := bootstrapConfig(ctx, appCtx, runCtx)
			if err != nil {
				return err
			}

			charts, err := runCtx.ChartFS.GetAllCharts()
			if err != nil {
				return err
			}

			collection, err := resolver.NewCollection(appCtx, charts)
			if err != nil {
				return err
			}

			configuredIntegrations, err := manager.ConfiguredIntegrations(ctx, cfg)
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
				return config.NewConfigMapManager(runCtx.Kube, appCtx.Name).
					Update(ctx, cfg)
			}

			return nil
		},
	}

	for _, mod := range manager.GetModules() {
		wrapper := manager.Integration(integrations.IntegrationName(mod.Name))
		sub := mod.Command(appCtx, runCtx, wrapper)
		cmd.AddCommand(api.NewRunner(sub).Cmd())
	}

	return cmd
}

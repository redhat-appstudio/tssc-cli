package subcmd

import (
	"context"
	"fmt"

	"github.com/redhat-appstudio/helmet/api"
	"github.com/redhat-appstudio/helmet/internal/config"
	"github.com/redhat-appstudio/helmet/internal/flags"
	"github.com/redhat-appstudio/helmet/internal/integrations"
	"github.com/redhat-appstudio/helmet/internal/resolver"
	"github.com/redhat-appstudio/helmet/internal/runcontext"

	"github.com/spf13/cobra"
)

// disableProductForIntegration disables the product that provides the active
// integration, if the integration secret exists and the product is currently
// enabled. Only the active integration is inspected; other integrations are
// not touched.
func disableProductForIntegration(
	ctx context.Context,
	appCtx *api.AppContext,
	runCtx *runcontext.RunContext,
	manager *integrations.Manager,
	cfg *config.Config,
	activeIntegration integrations.IntegrationName,
) error {
	// Check if THIS integration's secret was actually created.
	integration := manager.Integration(activeIntegration)
	exists, err := integration.Exists(ctx, cfg)
	if err != nil {
		return err
	}
	if !exists {
		return nil
	}

	// Find the product that provides this integration (if any).
	charts, err := runCtx.ChartFS.GetAllCharts()
	if err != nil {
		return err
	}
	collection, err := resolver.NewCollection(appCtx, charts)
	if err != nil {
		return err
	}
	productName := collection.GetProductNameForIntegration(
		string(activeIntegration))
	if productName == "" {
		return nil // no product provides this integration
	}

	spec, err := cfg.GetProduct(productName)
	if err != nil {
		return err
	}
	if !spec.Enabled {
		return nil // already disabled
	}

	spec.Enabled = false
	if err := cfg.SetProduct(productName, *spec); err != nil {
		return err
	}
	return config.NewConfigMapManager(runCtx.Kube, appCtx.Name).
		Update(ctx, cfg)
}

func NewIntegration(
	appCtx *api.AppContext,
	runCtx *runcontext.RunContext,
	manager *integrations.Manager,
	f *flags.Flags,
) *cobra.Command {
	cmd := &cobra.Command{
		Use: "integration <type>",
		Short: fmt.Sprintf(
			"Configures an external service provider for %s", appCtx.Name,
		),
		PersistentPostRunE: func(cmd *cobra.Command, _ []string) error {
			ctx := cmd.Context()

			// cmd is the child command (e.g., "acs", "quay").
			// cmd.Name() returns the integration name, matching the
			// IntegrationName used to register the module in Manager.
			activeIntegration := integrations.IntegrationName(cmd.Name())

			cfg, err := bootstrapConfig(ctx, appCtx, runCtx)
			if err != nil {
				return err
			}
			if err := disableProductForIntegration(
				ctx, appCtx, runCtx, manager, cfg, activeIntegration); err != nil {
				return err
			}
			if f.Verbose {
				_, err := fmt.Fprintln(cmd.OutOrStdout(), "Integration created successfully")
				return err
			}
			return nil
		},
	}

	for _, mod := range manager.GetModules() {
		wrapper := manager.Integration(integrations.IntegrationName(mod.Name))
		sub := mod.Command(appCtx, runCtx, wrapper)
		runner := api.NewRunner(sub)

		// Enforce: Cobra command name must match the integration module
		// name. When they differ, preserve the original as an alias for
		// backward compatibility.
		childCmd := runner.Cmd()
		if childCmd.Name() != mod.Name {
			childCmd.Aliases = append(childCmd.Aliases, childCmd.Name())
			childCmd.Use = mod.Name
		}

		cmd.AddCommand(childCmd)
	}

	return cmd
}

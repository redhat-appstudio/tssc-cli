package framework

import (
	"fmt"
	"os"

	"github.com/redhat-appstudio/helmet/pkg/api"
	"github.com/redhat-appstudio/helmet/pkg/chartfs"
	"github.com/redhat-appstudio/helmet/pkg/flags"
	"github.com/redhat-appstudio/helmet/pkg/integrations"
	"github.com/redhat-appstudio/helmet/pkg/k8s"
	"github.com/redhat-appstudio/helmet/pkg/mcptools"
	"github.com/redhat-appstudio/helmet/pkg/subcmd"

	"github.com/spf13/cobra"
)

// App represents the installer application runtime.
// It holds runtime dependencies and coordinates the execution of commands.
// Application metadata (name, version, etc.) is stored in AppCtx.
type App struct {
	AppCtx  *api.AppContext  // application metadata (single source of truth)
	ChartFS *chartfs.ChartFS // installer filesystem

	integrations       []api.IntegrationModule // supported integrations
	integrationManager *integrations.Manager   // integrations manager
	rootCmd            *cobra.Command          // root cobra instance
	flags              *flags.Flags            // global flags
	kube               *k8s.Kube               // kubernetes client

	mcpToolsBuilder   mcptools.MCPToolsBuilder // tools builder
	mcpImage          string                   // installer image
	installerTarball  []byte                   // embedded installer tarball
}

// Command exposes the Cobra command.
func (a *App) Command() *cobra.Command {
	return a.rootCmd
}

// Run is a shortcut Cobra's Execute method.
func (a *App) Run() error {
	return a.rootCmd.Execute()
}

// setupRootCmd instantiates the Cobra Root command with subcommand, description,
// Kubernetes API client instance and more.
func (a *App) setupRootCmd() error {
	short := a.AppCtx.Short
	if short == "" {
		short = a.AppCtx.Name + " installer"
	}

	a.rootCmd = &cobra.Command{
		Use:          a.AppCtx.Name,
		Short:        short,
		Long:         a.AppCtx.Long,
		SilenceUsage: true,
	}

	// Add persistent flags.
	a.flags.PersistentFlags(a.rootCmd.PersistentFlags())

	// Handle version flag and help.
	a.rootCmd.RunE = func(cmd *cobra.Command, args []string) error {
		if a.flags.Version {
			a.flags.ShowVersion(
				a.AppCtx.Name, a.AppCtx.Version, a.AppCtx.CommitID)
			return nil
		}
		return cmd.Help()
	}

	logger := a.flags.GetLogger(os.Stdout)

	// Loading informed integrations into the manager.
	a.integrationManager = integrations.NewManager()
	if err := a.integrationManager.LoadModules(
		a.AppCtx.Name, logger, a.kube, a.integrations,
	); err != nil {
		return fmt.Errorf("failed to load modules: %w", err)
	}

	// Register standard subcommands.
	a.rootCmd.AddCommand(subcmd.NewIntegration(
		a.AppCtx, logger, a.kube, a.ChartFS, a.integrationManager,
	))

	// Use default builder if none provided.
	mcpBuilder := a.mcpToolsBuilder
	if mcpBuilder == nil {
		mcpBuilder = subcmd.StandardMCPToolsBuilder()
	}

	// Validate MCP image is configured.
	if a.mcpImage == "" {
		return fmt.Errorf(
			"MCP server image not configured: use WithMCPImage() option")
	}

	// Other subcommands via api.Runner.
	subs := []api.SubCommand{
		subcmd.NewConfig(
			a.AppCtx,
			logger,
			a.flags,
			a.ChartFS,
			a.kube,
		),
		subcmd.NewDeploy(
			a.AppCtx,
			logger,
			a.flags,
			a.ChartFS,
			a.kube,
			a.integrationManager,
			a.installerTarball,
		),
		subcmd.NewInstaller(
			a.AppCtx,
			a.flags,
			a.installerTarball,
		),
		subcmd.NewMCPServer(
			a.AppCtx,
			a.flags,
			a.ChartFS,
			a.kube,
			a.integrationManager,
			mcpBuilder,
			a.mcpImage,
		),
		subcmd.NewTemplate(
			a.AppCtx,
			logger,
			a.flags,
			a.ChartFS,
			a.kube,
			a.installerTarball,
		),
		subcmd.NewTopology(
			a.AppCtx,
			logger,
			a.ChartFS,
			a.kube,
		),
	}
	for _, sub := range subs {
		a.rootCmd.AddCommand(api.NewRunner(sub).Cmd())
	}
	return nil
}

// NewApp creates a new installer application runtime.
// It automatically sets up the Cobra Root Command and standard subcommands.
//
// The appCtx parameter provides application metadata (name, version, etc.).
// The cfs parameter provides access to the installer filesystem (charts, config).
// Additional runtime options can be passed via functional options.
func NewApp(
	appCtx *api.AppContext,
	cfs *chartfs.ChartFS,
	opts ...Option,
) (*App, error) {
	app := &App{
		AppCtx:  appCtx,
		ChartFS: cfs,
		flags:   flags.NewFlags(),
	}

	for _, opt := range opts {
		opt(app)
	}

	// Initialize Kube client with flags
	app.kube = k8s.NewKube(app.flags)

	if err := app.setupRootCmd(); err != nil {
		return nil, err
	}

	return app, nil
}

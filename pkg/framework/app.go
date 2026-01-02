package framework

import (
	"fmt"
	"os"

	"github.com/redhat-appstudio/tssc-cli/pkg/api"
	"github.com/redhat-appstudio/tssc-cli/pkg/chartfs"
	"github.com/redhat-appstudio/tssc-cli/pkg/constants"
	"github.com/redhat-appstudio/tssc-cli/pkg/flags"
	"github.com/redhat-appstudio/tssc-cli/pkg/integrations"
	"github.com/redhat-appstudio/tssc-cli/pkg/k8s"
	"github.com/redhat-appstudio/tssc-cli/pkg/mcptools"
	"github.com/redhat-appstudio/tssc-cli/pkg/subcmd"

	"github.com/spf13/cobra"
)

// App represents the installer application.
type App struct {
	Name    string           // application name
	Version string           // application version
	Short   string           // short description
	Long    string           // long description
	ChartFS *chartfs.ChartFS // installer filesystem

	integrations       []api.IntegrationModule // supported integrations
	integrationManager *integrations.Manager   // integrations manager
	rootCmd            *cobra.Command          // root cobra instance
	flags              *flags.Flags            // global flags
	kube               *k8s.Kube               // kubernetes client

	mcpToolsBuilder mcptools.MCPToolsBuilder // tools builder
	mcpImage        string                   // installer image
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
	short := a.Short
	if short == "" {
		short = a.Name + " installer"
	}

	a.rootCmd = &cobra.Command{
		Use:          a.Name,
		Short:        short,
		Long:         a.Long,
		SilenceUsage: true,
	}

	// Add persistent flags.
	a.flags.PersistentFlags(a.rootCmd.PersistentFlags())

	// Handle version flag and help.
	a.rootCmd.RunE = func(cmd *cobra.Command, args []string) error {
		if a.flags.Version {
			a.flags.ShowVersion()
			return nil
		}
		return cmd.Help()
	}

	logger := a.flags.GetLogger(os.Stdout)

	// Loading informed integrations into the manager.
	a.integrationManager = integrations.NewManager()
	if err := a.integrationManager.LoadModules(
		a.Name, logger, a.kube, a.integrations,
	); err != nil {
		return fmt.Errorf("failed to load modules: %w", err)
	}

	// Register standard subcommands.
	a.rootCmd.AddCommand(subcmd.NewIntegration(
		a.Name, logger, a.kube, a.ChartFS, a.integrationManager,
	))

	// Use default builder if none provided
	mcpBuilder := a.mcpToolsBuilder
	if mcpBuilder == nil {
		mcpBuilder = subcmd.StandardMCPToolsBuilder()
	}

	// Determine MCP image - use configured value or compute default from constants
	mcpImage := a.mcpImage
	if mcpImage == "" {
		// Default image based on TSSC convention: image tagged with commit-id
		mcpImage = "quay.io/redhat-user-workloads/rhtap-shared-team-tenant/tssc-cli"
		if constants.CommitID == "" {
			mcpImage = fmt.Sprintf("%s:latest", mcpImage)
		} else {
			mcpImage = fmt.Sprintf("%s:%s", mcpImage, constants.CommitID)
		}
	}

	// Other subcommands via api.Runner.
	subs := []api.SubCommand{
		subcmd.NewConfig(logger, a.flags, a.ChartFS, a.kube),
		subcmd.NewDeploy(logger, a.flags, a.ChartFS, a.kube, a.integrationManager),
		subcmd.NewInstaller(a.flags),
		subcmd.NewMCPServer(
			a.Name,
			a.flags,
			a.ChartFS,
			a.kube,
			a.integrationManager,
			mcpBuilder,
			mcpImage,
		),
		subcmd.NewTemplate(logger, a.flags, a.ChartFS, a.kube),
		subcmd.NewTopology(logger, a.ChartFS, a.kube),
	}
	for _, sub := range subs {
		a.rootCmd.AddCommand(api.NewRunner(sub).Cmd())
	}
	return nil
}

// NewApp creates a new installer application. It automatically sets up the Cobra
// Root Command and standard subcommands (Config, Integration, Deploy).
func NewApp(name string, cfs *chartfs.ChartFS, opts ...Option) (*App, error) {
	app := &App{
		Name:    name,
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

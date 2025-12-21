package framework

import (
	"os"

	"github.com/redhat-appstudio/tssc-cli/pkg/api"
	"github.com/redhat-appstudio/tssc-cli/pkg/chartfs"
	"github.com/redhat-appstudio/tssc-cli/pkg/flags"
	"github.com/redhat-appstudio/tssc-cli/pkg/k8s"
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

	rootCmd *cobra.Command // root cobra instance
	flags   *flags.Flags   // global flags
	kube    *k8s.Kube      // kubernetes client
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
func (a *App) setupRootCmd() {
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

	// Register standard subcommands.
	a.rootCmd.AddCommand(subcmd.NewIntegration(logger, a.kube, a.ChartFS))

	// Other subcommands via api.Runner.
	subs := []api.SubCommand{
		subcmd.NewConfig(logger, a.flags, a.ChartFS, a.kube),
		subcmd.NewDeploy(logger, a.flags, a.ChartFS, a.kube),
		subcmd.NewInstaller(a.flags),
		subcmd.NewMCPServer(a.flags, a.ChartFS, a.kube),
		subcmd.NewTemplate(logger, a.flags, a.ChartFS, a.kube),
		subcmd.NewTopology(logger, a.ChartFS, a.kube),
	}
	for _, sub := range subs {
		a.rootCmd.AddCommand(api.NewRunner(sub).Cmd())
	}
}

// NewApp creates a new installer application. It automatically sets up the Cobra
// Root Command and standard subcommands (Config, Integration, Deploy).
func NewApp(name string, cfs *chartfs.ChartFS, opts ...Option) *App {
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

	app.setupRootCmd()

	return app
}

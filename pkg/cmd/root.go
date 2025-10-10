package cmd

import (
	"os"

	"github.com/redhat-appstudio/tssc-cli/pkg/chartfs"
	"github.com/redhat-appstudio/tssc-cli/pkg/constants"
	"github.com/redhat-appstudio/tssc-cli/pkg/flags"
	"github.com/redhat-appstudio/tssc-cli/pkg/k8s"
	"github.com/redhat-appstudio/tssc-cli/pkg/subcmd"

	"github.com/spf13/cobra"
)

// RootCmd is the root command.
type RootCmd struct {
	cmd   *cobra.Command // root command
	flags *flags.Flags   // global flags

	cfs  *chartfs.ChartFS // embedded filesystem
	kube *k8s.Kube        // kubernetes client
}

// Cmd exposes the root command, while instantiating the subcommand and their
// requirements.
func (r *RootCmd) Cmd() *cobra.Command {
	// Handle version flag
	r.cmd.RunE = func(cmd *cobra.Command, args []string) error {
		if r.flags.Version {
			r.flags.ShowVersion()
			return nil
		}
		return cmd.Help()
	}

	logger := r.flags.GetLogger(os.Stdout)

	r.cmd.AddCommand(subcmd.NewIntegration(logger, r.kube))

	for _, sub := range []subcmd.Interface{
		subcmd.NewConfig(logger, r.flags, r.cfs, r.kube),
		subcmd.NewDeploy(logger, r.flags, r.cfs, r.kube),
		subcmd.NewInstaller(r.flags),
		subcmd.NewMCPServer(r.flags, r.cfs, r.kube),
		subcmd.NewTemplate(logger, r.flags, r.cfs, r.kube),
		subcmd.NewTopology(logger, r.cfs, r.kube),
	} {
		r.cmd.AddCommand(subcmd.NewRunner(sub).Cmd())
	}
	return r.cmd
}

// NewRootCmd instantiates the root command, setting up the global flags.
func NewRootCmd() (*RootCmd, error) {
	f := flags.NewFlags()

	cfs, err := chartfs.NewChartFSForCWD()
	if err != nil {
		return nil, err
	}

	r := &RootCmd{
		flags: f,
		cmd: &cobra.Command{
			Use:          constants.AppName,
			Short:        "Trusted Software Supply Chain CLI",
			SilenceUsage: true,
		},
		cfs:  cfs,
		kube: k8s.NewKube(f),
	}
	p := r.cmd.PersistentFlags()
	r.flags.PersistentFlags(p)
	return r, nil
}

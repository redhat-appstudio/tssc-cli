package subcmd

import (
	"fmt"
	"log/slog"

	"github.com/redhat-appstudio/rhtap-cli/pkg/config"
	"github.com/redhat-appstudio/rhtap-cli/pkg/flags"
	"github.com/redhat-appstudio/rhtap-cli/pkg/installer"
	"github.com/redhat-appstudio/rhtap-cli/pkg/k8s"

	"github.com/spf13/cobra"
)

// Deploy is the deploy subcommand.
type Deploy struct {
	cmd    *cobra.Command // cobra command
	logger *slog.Logger   // application logger
	flags  *flags.Flags   // global flags
	cfg    *config.Config // installer configuration
	kube   *k8s.Kube      // kubernetes client

	valuesTmplPath string // path to the values template file
}

var _ Interface = &Deploy{}

const deployDesc = `
Deploys the RHTAP platform components. The installer looks the the informed
configuration to identify the features to be installed, and the dependencies to be
resolved.

The deployment configuration file describes the sequence of Helm charts to be
applied, on the attribute 'rhtapCLI.dependencies[]'.

The platform configuration is rendered from the values template file
(--values-template), this configuration payload is given to all Helm charts.

The installer resources are embedded in the executable, these resources are
employed by default, to use local files, set the '--embedded' flag to false.
`

// Cmd exposes the cobra instance.
func (d *Deploy) Cmd() *cobra.Command {
	return d.cmd
}

// log logger with contextual information.
func (d *Deploy) log() *slog.Logger {
	return d.flags.LoggerWith(
		d.logger.With("values-template", d.valuesTmplPath))
}

// Complete verifies the object is complete.
func (d *Deploy) Complete(_ []string) error {
	return nil
}

// Validate asserts the requirements to start the deployment are in place.
func (d *Deploy) Validate() error {
	return k8s.EnsureOpenShiftProject(
		d.cmd.Context(),
		d.log(),
		d.kube,
		d.cfg.Installer.Namespace,
	)
}

// Run deploys the enabled dependencies listed on the configuration.
func (d *Deploy) Run() error {
	cfs, err := newChartFS(d.logger, d.flags, d.cfg)
	if err != nil {
		return err
	}

	d.log().Debug("Reading values template file")
	valuesTmpl, err := cfs.ReadFile(d.valuesTmplPath)
	if err != nil {
		return fmt.Errorf("failed to read values template file: %w", err)
	}

	// Installing each Helm Chart dependency from the configuration, only
	// selecting the Helm Charts that are enabled.
	d.log().Debug("Installing dependencies...")
	deps := d.cfg.GetEnabledDependencies(d.log())
	for ix, dep := range deps {
		fmt.Printf("\n\n############################################################\n")
		fmt.Printf("# [%d/%d] Deploying '%s' in '%s'.\n", ix+1, len(deps), dep.Chart, dep.Namespace)
		fmt.Printf("############################################################\n")

		i := installer.NewInstaller(d.log(), d.flags, d.kube, cfs, &dep)

		err := i.SetValues(d.cmd.Context(), &d.cfg.Installer, string(valuesTmpl))
		if err != nil {
			return err
		}
		if d.flags.Debug {
			i.PrintRawValues()
		}

		if err := i.RenderValues(); err != nil {
			return err
		}
		if d.flags.Debug {
			i.PrintValues()
		}

		if err = i.Install(d.cmd.Context()); err != nil {
			return err
		}
		fmt.Printf("############################################################\n\n")
	}

	fmt.Printf("Deployment complete!\n")
	return nil
}

// NewDeploy instantiates the deploy subcommand.
func NewDeploy(
	logger *slog.Logger,
	f *flags.Flags,
	cfg *config.Config,
	kube *k8s.Kube,
) Interface {
	d := &Deploy{
		cmd: &cobra.Command{
			Use:          "deploy",
			Short:        "Rollout RHTAP platform components",
			Long:         deployDesc,
			SilenceUsage: true,
		},
		logger: logger.WithGroup("deploy"),
		flags:  f,
		cfg:    cfg,
		kube:   kube,
	}
	flags.SetValuesTmplFlag(d.cmd.PersistentFlags(), &d.valuesTmplPath)
	return d
}

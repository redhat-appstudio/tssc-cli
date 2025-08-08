package subcmd

import (
	"fmt"
	"log/slog"
	"strings"

	"github.com/redhat-appstudio/tssc-cli/pkg/chartfs"
	"github.com/redhat-appstudio/tssc-cli/pkg/config"
	"github.com/redhat-appstudio/tssc-cli/pkg/flags"
	"github.com/redhat-appstudio/tssc-cli/pkg/installer"
	"github.com/redhat-appstudio/tssc-cli/pkg/k8s"
	"github.com/redhat-appstudio/tssc-cli/pkg/printer"
	"github.com/redhat-appstudio/tssc-cli/pkg/resolver"

	"github.com/spf13/cobra"
)

// Deploy is the deploy subcommand.
type Deploy struct {
	cmd    *cobra.Command   // cobra command
	logger *slog.Logger     // application logger
	flags  *flags.Flags     // global flags
	cfg    *config.Config   // installer configuration
	cfs    *chartfs.ChartFS // embedded filesystem
	kube   *k8s.Kube        // kubernetes client

	collection         *resolver.Collection // chart collection
	chartPath          string               // single chart path
	valuesTemplatePath string               // values template file path
}

var _ Interface = &Deploy{}

const deployDesc = `
Deploys the TSSC platform components.

It should only be used to for experimental deployments. Production
deployments are not supported.

The installer looks at the configuration to identify the products to be
installed, and the dependencies to be resolved.

The deployment configuration file describes the sequence of Helm charts to be
applied, on the attribute 'tssc.dependencies[]'.

The platform configuration is rendered from the values template file
(--values-template), this configuration payload is given to all Helm charts.

The installer resources are embedded in the executable, these resources are
employed by default.

A single chart can be deployed by specifying its path. E.g.:
	tssc deploy charts/tssc-openshift
`

// Cmd exposes the cobra instance.
func (d *Deploy) Cmd() *cobra.Command {
	return d.cmd
}

// log logger with contextual information.
func (d *Deploy) log() *slog.Logger {
	return d.flags.LoggerWith(
		d.logger.With(flags.ValuesTemplateFlag, d.valuesTemplatePath))
}

// Complete verifies the object is complete.
func (d *Deploy) Complete(args []string) error {
	// Load all charts from the embedded filesystem, or from a local directory.
	charts, err := d.cfs.GetAllCharts()
	if err != nil {
		return err
	}
	// Create a new chart collection from the loaded charts.
	if d.collection, err = resolver.NewCollection(charts); err != nil {
		return err
	}
	// Load the installer configuration from the cluster.
	if d.cfg, err = bootstrapConfig(d.cmd.Context(), d.kube); err != nil {
		return err
	}
	if len(args) == 1 {
		d.chartPath = args[0]
	}
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
	printer.Disclaimer()

	d.log().Debug("Reading values template file")
	valuesTmpl, err := d.cfs.ReadFile(d.valuesTemplatePath)
	if err != nil {
		return fmt.Errorf("failed to read values template file: %w", err)
	}

	d.log().Debug("Resolving dependencies...")
	topology := resolver.NewTopology()
	r := resolver.NewResolver(d.cfg, d.collection, topology)
	if err := r.Resolve(); err != nil {
		return err
	}

	var deps []config.Dependency
	if d.chartPath == "" {
		d.log().Debug("Installing all dependencies...")
		deps = topology.GetDependencies()
	} else {
		d.log().Debug("Installing a single Helm chart...")
		dep, err := topology.GetDependencyForChart(d.chartPath)
		if err != nil {
			return err
		}
		deps = append(deps, *dep)
	}

	for index, dep := range deps {
		fmt.Printf("\n\n%s\n", strings.Repeat("#", 60))
		fmt.Printf(
			"# [%d/%d] Deploying '%s' in '%s'.\n",
			index+1,
			len(deps),
			dep.Chart.Name(),
			dep.Namespace,
		)
		fmt.Printf("%s\n", strings.Repeat("#", 60))

		i := installer.NewInstaller(d.log(), d.flags, d.kube, &dep)

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
		// Cleaning up temporary resources.
		if err = k8s.RetryDeleteResources(
			d.cmd.Context(),
			d.kube,
			d.cfg.Installer.Namespace,
		); err != nil {
			d.log().Debug(err.Error())
		}
		fmt.Printf("%s\n", strings.Repeat("#", 60))
	}

	fmt.Printf("Deployment complete!\n")
	return nil
}

// NewDeploy instantiates the deploy subcommand.
func NewDeploy(
	logger *slog.Logger,
	f *flags.Flags,
	cfs *chartfs.ChartFS,
	kube *k8s.Kube,
) Interface {
	d := &Deploy{
		cmd: &cobra.Command{
			Use:          "deploy [chart]",
			Short:        "Rollout TSSC platform components",
			Long:         deployDesc,
			SilenceUsage: true,
		},
		logger:    logger.WithGroup("deploy"),
		flags:     f,
		cfs:       cfs,
		kube:      kube,
		chartPath: "",
	}
	flags.SetValuesTmplFlag(d.cmd.PersistentFlags(), &d.valuesTemplatePath)
	return d
}

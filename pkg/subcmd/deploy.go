package subcmd

import (
	"fmt"
	"log/slog"
	"os"

	"github.com/otaviof/rhtap-installer-cli/pkg/config"
	"github.com/otaviof/rhtap-installer-cli/pkg/deployer"
	"github.com/otaviof/rhtap-installer-cli/pkg/engine"
	"github.com/otaviof/rhtap-installer-cli/pkg/flags"
	"github.com/otaviof/rhtap-installer-cli/pkg/k8s"

	"github.com/spf13/cobra"
	"helm.sh/helm/v3/pkg/chartutil"
)

// Deploy is the deploy sub-command.
type Deploy struct {
	logger *slog.Logger       // application logger
	cmd    *cobra.Command     // cobra command
	flags  *flags.Flags       // global flags
	cfg    *config.ConfigSpec // installer configuration
	kube   *k8s.Kube          // kubernetes client

	valuesTemplatePath string
}

var _ Interface = &Deploy{}

func (d *Deploy) Cmd() *cobra.Command {
	return d.cmd
}

func (d *Deploy) log() *slog.Logger {
	return d.logger.With(
		"values-template", d.valuesTemplatePath,
		"dry-run", d.flags.DryRun,
		"debug", d.flags.Debug,
	)
}

func (d *Deploy) Complete(_ []string) error {
	if d.cfg == nil {
		return fmt.Errorf("configuration is not informed")
	}
	if d.kube == nil {
		return fmt.Errorf("kubernetes client is not informed")
	}
	return nil
}

func (d *Deploy) Validate() error {
	d.log().Debug("Verifying Kubernetes client connection...")
	if err := d.kube.Connected(); err != nil {
		return err
	}
	d.log().Debug("Ensure the OpenShift project for RHTAP installer is created")
	return deployer.EnsureOpenShiftProject(d.log(), d.kube, d.cfg.Namespace)
}

func (d *Deploy) Run() error {
	d.log().Debug("Loading values template file")
	valuesTemplatePayload, err := os.ReadFile(d.valuesTemplatePath)
	if err != nil {
		return err
	}

	d.log().Debug("Preparing values template context")
	variables := engine.NewVariables()
	if err := variables.SetInstaller(d.cfg); err != nil {
		return err
	}

	eng := engine.NewEngine(string(valuesTemplatePayload))

	for _, dep := range d.cfg.Dependencies {
		logger := d.log().With(
			"dependency-chart", dep.Chart,
			"dependency-namespace", dep.Namespace,
		)

		hc, err := deployer.NewHelm(logger, d.flags, d.kube, &dep)
		if err != nil {
			return err
		}

		logger.Debug("Rendering values from template")
		valuesBytes, err := eng.Render(variables)
		if err != nil {
			return err
		}

		logger.Debug("Preparing rendered values for Helm installation")
		values, err := chartutil.ReadValues(valuesBytes)
		if err != nil {
			return err
		}

		logger.Debug("Installing the Helm chart")
		if err = hc.Install(values); err != nil {
			return err
		}
		logger.Debug("Verifying the Helm chart release")
		if err = hc.Verify(); err != nil {
			return err
		}
		logger.Info("Helm chart installed!")
	}
	return nil
}

func NewDeploy(
	logger *slog.Logger,
	f *flags.Flags,
	cfg *config.ConfigSpec,
	kube *k8s.Kube,
) Interface {
	d := &Deploy{
		logger: logger.WithGroup("deploy"),
		cmd: &cobra.Command{
			Use:          "deploy",
			Short:        "Deploys the RHTAP components",
			SilenceUsage: true,
		},
		flags: f,
		cfg:   cfg,
		kube:  kube,
	}
	flags.SetValuesTmplFlag(d.cmd.PersistentFlags(), &d.valuesTemplatePath)
	return d
}

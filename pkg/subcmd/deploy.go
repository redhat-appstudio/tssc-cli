package subcmd

import (
	"fmt"
	"log/slog"
	"os"

	"github.com/redhat-appstudio/rhtap-cli/pkg/config"
	"github.com/redhat-appstudio/rhtap-cli/pkg/deployer"
	"github.com/redhat-appstudio/rhtap-cli/pkg/engine"
	"github.com/redhat-appstudio/rhtap-cli/pkg/flags"
	"github.com/redhat-appstudio/rhtap-cli/pkg/hooks"
	"github.com/redhat-appstudio/rhtap-cli/pkg/k8s"

	"github.com/spf13/cobra"
	"helm.sh/helm/v3/pkg/chartutil"
)

// Deploy is the deploy subcommand.
type Deploy struct {
	cmd    *cobra.Command // cobra command
	logger *slog.Logger   // application logger
	flags  *flags.Flags   // global flags
	cfg    *config.Spec   // installer configuration
	kube   *k8s.Kube      // kubernetes client

	valuesTemplatePath string // path to the values template file
}

var _ Interface = &Deploy{}

const deployDesc = `
Deploys the RHTAP platform components. The installer looks the the informed
configuration to identify the features to be installed, and the dependencies to be
resolved.

The deployment configuration file describes the sequence of Helm charts to be
applied, on the attribute 'rhtapInstallerCLI.dependencies[]'.

The platform configuration is rendered from the values template file
(--values-template), this configuration payload is given to all Helm charts.
`

// Cmd exposes the cobra instance.
func (d *Deploy) Cmd() *cobra.Command {
	return d.cmd
}

// log logger with contextual information.
func (d *Deploy) log() *slog.Logger {
	return d.flags.LoggerWith(
		d.logger.With("values-template", d.valuesTemplatePath))
}

// Complete verifies the object is complete.
func (d *Deploy) Complete(_ []string) error {
	if d.cfg == nil {
		return fmt.Errorf("configuration is not informed")
	}
	if d.kube == nil {
		return fmt.Errorf("kubernetes client is not informed")
	}
	return nil
}

// Validate asserts the requirements to start the deployment are in place.
func (d *Deploy) Validate() error {
	return k8s.EnsureOpenShiftProject(
		d.cmd.Context(),
		d.log(),
		d.kube,
		d.cfg.Namespace,
	)
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
	if err := variables.SetOpenShift(d.cmd.Context(), d.kube); err != nil {
		return err
	}

	eng := engine.NewEngine(d.kube, string(valuesTemplatePayload))

	for _, dep := range d.cfg.Dependencies {
		logger := dep.LoggerWith(d.log())

		hc, err := deployer.NewHelm(logger, d.flags, d.kube, dep)
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

		hook := hooks.NewHooks(dep)
		logger.Debug("Running pre-deploy hook script...")
		if err = hook.PreDeploy(values); err != nil {
			return err
		}

		// Performing the installation, or upgrade, of the Helm chart dependency,
		// using the values rendered before hand.
		logger.Debug("Installing the Helm chart")
		if err = hc.Install(values); err != nil {
			return err
		}
		// Verifying if the instaltion was successful, by running the Helm chart
		// tests interactively.
		logger.Debug("Verifying the Helm chart release")
		if err = hc.Verify(); err != nil {
			return err
		}

		logger.Debug("Running post-deploy hook script...")
		if err = hook.PostDeploy(values); err != nil {
			return err
		}
		logger.Info("Helm chart installed!")
	}
	return nil
}

func NewDeploy(
	logger *slog.Logger,
	f *flags.Flags,
	cfg *config.Spec,
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
	flags.SetValuesTmplFlag(d.cmd.PersistentFlags(), &d.valuesTemplatePath)
	return d
}

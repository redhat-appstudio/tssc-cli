package installer

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	"github.com/redhat-appstudio/tssc-cli/pkg/config"
	"github.com/redhat-appstudio/tssc-cli/pkg/deployer"
	"github.com/redhat-appstudio/tssc-cli/pkg/engine"
	"github.com/redhat-appstudio/tssc-cli/pkg/flags"
	"github.com/redhat-appstudio/tssc-cli/pkg/hooks"
	"github.com/redhat-appstudio/tssc-cli/pkg/k8s"
	"github.com/redhat-appstudio/tssc-cli/pkg/monitor"
	"github.com/redhat-appstudio/tssc-cli/pkg/printer"
	"github.com/redhat-appstudio/tssc-cli/pkg/resolver"

	"helm.sh/helm/v3/pkg/chartutil"
)

// Installer represents the "helm install" using its APIs, this component deploys
// the informed dependency on the pre-configured namespace.
type Installer struct {
	logger *slog.Logger         // application logger
	flags  *flags.Flags         // global flags
	kube   *k8s.Kube            // kubernetes client
	dep    *resolver.Dependency // dependency to install

	valuesBytes []byte           // rendered values
	values      chartutil.Values // helm chart values
}

// SetValues prepares the values template for the Helm chart installation.
func (i *Installer) SetValues(
	ctx context.Context,
	cfg *config.Spec,
	valuesTmpl string,
) error {
	i.logger.Debug("Preparing values template context")
	variables := engine.NewVariables()
	err := variables.SetInstaller(cfg)
	if err != nil {
		return err
	}
	if err = variables.SetOpenShift(ctx, i.kube); err != nil {
		return err
	}

	i.logger.Debug("Rendering values template")
	i.valuesBytes, err = engine.NewEngine(i.kube, valuesTmpl).Render(variables)
	return err
}

// PrintRawValues prints the raw values template to the console.
func (i *Installer) PrintRawValues() {
	i.logger.Debug("Showing raw results of rendered values template")
	fmt.Printf("#\n# Values (Raw)\n#\n\n%s\n", i.valuesBytes)
}

// RenderValues parses the values template and prepares the Helm chart values.
func (i *Installer) RenderValues() error {
	if i.valuesBytes == nil {
		return fmt.Errorf("values not set")
	}

	i.logger.Debug("Preparing rendered values for Helm installation")
	var err error
	i.values, err = chartutil.ReadValues(i.valuesBytes)
	return err
}

// PrintValues prints the parsed values to the console.
func (i *Installer) PrintValues() {
	i.logger.Debug("Showing parsed values")
	printer.ValuesPrinter("Values", i.values)
}

// Install performs the installation of the Helm chart, including the pre and post
// hooks execution.
func (i *Installer) Install(ctx context.Context) error {
	if i.values == nil {
		return fmt.Errorf("values not set")
	}

	i.logger.Debug("Loading Helm client for dependency and namespace")
	hc, err := deployer.NewHelm(
		i.logger,
		i.flags,
		i.kube,
		i.dep.Namespace(),
		i.dep.Chart(),
	)
	if err != nil {
		return err
	}

	hook := hooks.NewHooks(i.dep, os.Stdout, os.Stderr)
	if !i.flags.DryRun {
		i.logger.Debug("Running pre-deploy hook script...")
		if err = hook.PreDeploy(i.values); err != nil {
			return err
		}
	} else {
		i.logger.Debug("Skipping pre-deploy hook script (dry-run)")
	}

	// Performing the installation, or upgrade, of the Helm chart dependency,
	// using the values rendered before hand.
	i.logger.Debug("Installing the Helm chart")
	if err = hc.Deploy(i.values); err != nil {
		return err
	}
	// Verifying if the installation was successful, by running the Helm chart
	// tests interactively.
	i.logger.Debug("Verifying the Helm chart release")
	if err = hc.VerifyWithRetry(); err != nil {
		return err
	}

	if !i.flags.DryRun {
		m := monitor.NewMonitor(i.logger, i.kube)
		i.logger.Debug("Collecting resources for monitoring...")
		if err = hc.VisitReleaseResources(ctx, m); err != nil {
			return err
		}
		i.logger.Debug("Monitoring the Helm chart release...")
		if err = m.Watch(i.flags.Timeout); err != nil {
			return err
		}
		i.logger.Debug("Monitoring completed, release is successful!")

		i.logger.Debug("Running post-deploy hook script...")
		if err = hook.PostDeploy(i.values); err != nil {
			return err
		}
	} else {
		i.logger.Debug("Skipping monitoring and post-deploy hook (dry-run)")
	}

	i.logger.Info("Helm chart installed!")
	return nil
}

// NewInstaller instantiates a new installer for the given dependency.
func NewInstaller(
	logger *slog.Logger,
	f *flags.Flags,
	kube *k8s.Kube,
	dep *resolver.Dependency,
) *Installer {
	return &Installer{
		logger: dep.LoggerWith(logger),
		flags:  f,
		kube:   kube,
		dep:    dep,
	}
}

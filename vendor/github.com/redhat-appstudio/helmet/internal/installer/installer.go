package installer

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	"github.com/redhat-appstudio/helmet/internal/config"
	"github.com/redhat-appstudio/helmet/internal/deployer"
	"github.com/redhat-appstudio/helmet/internal/engine"
	"github.com/redhat-appstudio/helmet/internal/flags"
	"github.com/redhat-appstudio/helmet/internal/hooks"
	"github.com/redhat-appstudio/helmet/internal/k8s"
	"github.com/redhat-appstudio/helmet/internal/monitor"
	"github.com/redhat-appstudio/helmet/internal/printer"
	"github.com/redhat-appstudio/helmet/internal/resolver"

	"helm.sh/helm/v3/pkg/chartutil"
)

// Installer represents the "helm install" using its APIs, this component deploys
// the informed dependency on the pre-configured namespace.
type Installer struct {
	logger *slog.Logger         // application logger
	flags  *flags.Flags         // global flags
	kube   *k8s.Kube            // kubernetes client
	dep    *resolver.Dependency // dependency to install

	valuesBytes      []byte           // rendered values
	values           chartutil.Values // helm chart values
	installerTarball []byte           // embedded installer tarball
}

// SetValues prepares the values template for the Helm chart installation.
func (i *Installer) SetValues(
	ctx context.Context,
	cfg *config.Config,
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
	if err = hc.Deploy(ctx, i.values); err != nil {
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
	installerTarball []byte,
) *Installer {
	return &Installer{
		logger:           dep.LoggerWith(logger),
		flags:            f,
		kube:             kube,
		dep:              dep,
		installerTarball: installerTarball,
	}
}

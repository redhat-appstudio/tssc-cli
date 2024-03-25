package deployer

import (
	"fmt"
	"log/slog"
	"os"

	"github.com/otaviof/rhtap-installer-cli/pkg/config"
	"github.com/otaviof/rhtap-installer-cli/pkg/flags"
	"github.com/otaviof/rhtap-installer-cli/pkg/k8s"
	"github.com/otaviof/rhtap-installer-cli/pkg/printer"

	"github.com/pkg/errors"
	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/chart/loader"
	"helm.sh/helm/v3/pkg/chartutil"
	"helm.sh/helm/v3/pkg/registry"
	"helm.sh/helm/v3/pkg/release"
	"helm.sh/helm/v3/pkg/storage/driver"
)

// Helm represents the Helm support for the installer. It's responsible for
// running the Helm related actions.
type Helm struct {
	logger *slog.Logger // application logger
	flags  *flags.Flags // global flags

	dependency config.Dependency     // helm chart coordinates
	chart      *chart.Chart          // helm chart instance
	actionCfg  *action.Configuration // helm action configuration
}

// ErrInstallFailed when the Helm chart installation fails.
var ErrInstallFailed = errors.New("install failed")

// ErrUpgradeFailed when the Helm chart upgrade fails.
var ErrUpgradeFailed = errors.New("upgrade failed")

// log logger with contextual information.
func (h *Helm) log() *slog.Logger {
	return h.dependency.LoggerWith(h.logger.With("type", "helm"))
}

// printRelease prints the Helm release information.
func (h *Helm) printRelease(rel *release.Release) {
	printer.HelmReleasePrinter(rel)
	if h.flags.Debug {
		printer.HelmExtendedReleasePrinter(rel)
	}
}

// helmInstall equivalent to "helm install" command.
func (h *Helm) helmInstall(vals chartutil.Values) (*release.Release, error) {
	c := action.NewInstall(h.actionCfg)
	c.GenerateName = false
	c.Namespace = h.dependency.Namespace
	c.ReleaseName = h.chart.Name()
	c.Timeout = h.flags.Timeout

	c.DryRun = h.flags.DryRun
	c.ClientOnly = h.flags.DryRun
	if h.flags.DryRun {
		c.DryRunOption = "client"
	}

	ctx := backgroundContext(func() {
		h.log().Warn("Release installation has been cancelled.")
	})

	rel, err := c.RunWithContext(ctx, h.chart, vals)
	if err != nil {
		return nil, fmt.Errorf("%w: %s", ErrInstallFailed, err.Error())
	}
	return rel, nil
}

// helmUpgrade equivalent to "helm upgrade" command.
func (h *Helm) helmUpgrade(vals chartutil.Values) (*release.Release, error) {
	c := action.NewUpgrade(h.actionCfg)
	c.Namespace = h.dependency.Namespace
	c.Timeout = h.flags.Timeout

	c.DryRun = h.flags.DryRun
	if h.flags.DryRun {
		c.DryRunOption = "client"
	}

	ctx := backgroundContext(func() {
		h.log().Warn("Release upgrade has been cancelled.")
	})

	rel, err := c.RunWithContext(ctx, h.chart.Name(), h.chart, vals)
	if err != nil {
		return nil, fmt.Errorf("%w: %s", ErrUpgradeFailed, err.Error())
	}
	return rel, err
}

// Install installs the Helm chart (Dependency) on the cluster. It checks if the
// release is already installed in order to use the proper helm-client (action).
func (h *Helm) Install(vals chartutil.Values) error {
	c := action.NewHistory(h.actionCfg)
	c.Max = 1

	var rel *release.Release
	h.log().Debug("Checking if release exists on the cluster")
	var err error
	if _, err = c.Run(h.chart.Name()); errors.Is(err, driver.ErrReleaseNotFound) {
		h.log().Info("Installing Helm Chart...")
		rel, err = h.helmInstall(vals)
	} else {
		h.log().Info("Upgrading Helm Chart...")
		rel, err = h.helmUpgrade(vals)
	}
	if err != nil {
		return err
	}
	h.printRelease(rel)
	return nil
}

// Verify equivalent to "helm test", it checks whether the release is correctly
// deployed by running chart tests and waiting for successful result.
func (h *Helm) Verify() error {
	if h.flags.DryRun {
		h.log().Debug("Dry-run mode enabled, skipping verification")
		return nil
	}

	h.log().Debug("Verifying the release...")
	c := action.NewReleaseTesting(h.actionCfg)
	c.Namespace = h.dependency.Namespace

	_, err := c.Run(h.chart.Name())
	if err != nil {
		return err
	}
	h.log().Info("Release verified!")
	return nil
}

// NewHelm creates a new Helm instance, setting up the Helm action configuration
// to be used on subsequent Helm interactions. The Helm instance is bound to a
// single Helm Chart (Dependency).
func NewHelm(
	logger *slog.Logger,
	f *flags.Flags,
	kube *k8s.Kube,
	dep config.Dependency,
) (*Helm, error) {
	actionCfg := new(action.Configuration)
	getter := kube.RESTClientGetter(dep.Namespace)
	driver := os.Getenv("HELM_DRIVER")

	loggerFn := func(format string, v ...interface{}) {
		logger.WithGroup("helm-cli").Debug(fmt.Sprintf(format, v...))
	}
	err := actionCfg.Init(getter, dep.Namespace, driver, loggerFn)
	if err != nil {
		return nil, err
	}

	actionCfg.RegistryClient, err = registry.NewClient(
		registry.ClientOptDebug(true))
	if err != nil {
		return nil, err
	}

	chart, err := loader.Load(dep.Chart)
	if err != nil {
		return nil, err
	}

	return &Helm{
		logger:     logger,
		flags:      f,
		chart:      chart,
		actionCfg:  actionCfg,
		dependency: dep,
	}, nil
}

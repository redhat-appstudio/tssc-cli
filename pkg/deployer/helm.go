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

	dep       *config.Dependency
	chart     *chart.Chart
	actionCfg *action.Configuration
}

// ErrInstallFailed when the Helm chart installation fails.
var ErrInstallFailed = errors.New("install failed")

// ErrUpgradeFailed when the Helm chart upgrade fails.
var ErrUpgradeFailed = errors.New("upgrade failed")

func (h *Helm) printRelease(rel *release.Release) {
	printer.HelmReleasePrinter(rel)
	if h.flags.Debug {
		printer.HelmExtendedReleasePrinter(rel)
	}
}

func (h *Helm) helmInstall(vals chartutil.Values) (*release.Release, error) {
	c := action.NewInstall(h.actionCfg)
	c.DryRun = h.flags.DryRun
	c.GenerateName = false
	c.Namespace = h.dep.Namespace
	c.ReleaseName = h.chart.Name()
	c.Wait = true

	ctx := backgroundContext(func() {
		h.logger.Warn("Release installation has been cancelled.")
	})

	rel, err := c.RunWithContext(ctx, h.chart, vals)
	if err != nil {
		return nil, fmt.Errorf("%w: %s", ErrInstallFailed, err.Error())
	}
	return rel, nil
}

func (h *Helm) helmUpgrade(vals chartutil.Values) (*release.Release, error) {
	c := action.NewUpgrade(h.actionCfg)
	c.DryRun = h.flags.DryRun
	c.Namespace = h.dep.Namespace
	c.Wait = true

	ctx := backgroundContext(func() {
		h.logger.Warn("Release upgrade has been cancelled.")
	})

	rel, err := c.RunWithContext(ctx, h.chart.Name(), h.chart, vals)
	if err != nil {
		return nil, fmt.Errorf("%w: %s", ErrUpgradeFailed, err.Error())
	}
	return rel, err
}

func (h *Helm) Install(vals chartutil.Values) error {
	c := action.NewHistory(h.actionCfg)
	c.Max = 1

	rel := &release.Release{}
	h.logger.Debug("Checking if release exists on the cluster")
	var err error
	if _, err = c.Run(h.chart.Name()); err == driver.ErrReleaseNotFound {
		h.logger.Info("Installing Helm Chart...")
		rel, err = h.helmInstall(vals)
	} else {
		h.logger.Info("Upgrading Helm Chart...")
		rel, err = h.helmUpgrade(vals)
	}
	if err != nil {
		return err
	}
	h.printRelease(rel)
	return nil
}

func (h *Helm) Verify() error {
	if h.flags.DryRun {
		h.logger.Debug("Dry-run mode enabled, skipping verification")
		return nil
	}

	h.logger.Debug("Verifying the release...")
	c := action.NewReleaseTesting(h.actionCfg)
	c.Namespace = h.dep.Namespace
	rel, err := c.Run(h.chart.Name())
	if err != nil {
		return err
	}

	h.printRelease(rel)
	return nil
}

func NewHelm(
	logger *slog.Logger,
	f *flags.Flags,
	kube *k8s.Kube,
	dep *config.Dependency,
) (*Helm, error) {
	helmLogger := logger.With(
		"type", "helm",
		"helm-chart", dep.Chart,
		"helm-namespace", dep.Namespace,
	)

	actionCfg := new(action.Configuration)

	loggerFn := func(format string, v ...interface{}) {
		helmLogger.Debug(fmt.Sprintf(format, v...))
	}

	getter := kube.RESTClientGetter(dep.Namespace)
	driver := os.Getenv("HELM_DRIVER")

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
		logger:    helmLogger,
		flags:     f,
		chart:     chart,
		actionCfg: actionCfg,
		dep:       dep,
	}, nil
}

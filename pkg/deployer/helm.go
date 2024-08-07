package deployer

import (
	"bytes"
	"context"
	"fmt"
	"log/slog"
	"os"

	"github.com/redhat-appstudio/rhtap-cli/pkg/flags"
	"github.com/redhat-appstudio/rhtap-cli/pkg/k8s"
	"github.com/redhat-appstudio/rhtap-cli/pkg/monitor"
	"github.com/redhat-appstudio/rhtap-cli/pkg/printer"

	"github.com/pkg/errors"
	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/chartutil"
	"helm.sh/helm/v3/pkg/registry"
	"helm.sh/helm/v3/pkg/release"
	"helm.sh/helm/v3/pkg/storage/driver"
	"k8s.io/cli-runtime/pkg/resource"
)

// Helm represents the Helm support for the installer. It's responsible for
// running the Helm related actions.
type Helm struct {
	logger *slog.Logger // application logger
	flags  *flags.Flags // global flags

	chart     *chart.Chart          // helm chart instance
	namespace string                // kubernetes namespace
	actionCfg *action.Configuration // helm action configuration

	release *release.Release // helm chart release
}

// ErrInstallFailed when the Helm chart installation fails.
var ErrInstallFailed = errors.New("install failed")

// ErrUpgradeFailed when the Helm chart upgrade fails.
var ErrUpgradeFailed = errors.New("upgrade failed")

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
	c.Namespace = h.namespace
	c.ReleaseName = h.chart.Name()
	c.Timeout = h.flags.Timeout

	c.DryRun = h.flags.DryRun
	c.ClientOnly = h.flags.DryRun
	if h.flags.DryRun {
		c.DryRunOption = "client"
	}

	ctx := backgroundContext(func() {
		h.logger.Warn("Release installation has been cancelled.")
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
	c.Namespace = h.namespace
	c.Timeout = h.flags.Timeout

	c.DryRun = h.flags.DryRun
	if h.flags.DryRun {
		c.DryRunOption = "client"
	}

	ctx := backgroundContext(func() {
		h.logger.Warn("Release upgrade has been cancelled.")
	})

	rel, err := c.RunWithContext(ctx, h.chart.Name(), h.chart, vals)
	if err != nil {
		return nil, fmt.Errorf("%w: %s", ErrUpgradeFailed, err.Error())
	}
	return rel, err
}

// Deploy deploys the Helm chart (Dependency) on the cluster. It checks if the
// release is already installed in order to use the proper helm-client (action).
func (h *Helm) Deploy(vals chartutil.Values) error {
	c := action.NewHistory(h.actionCfg)
	c.Max = 1

	h.logger.Debug("Checking if release exists on the cluster")
	var err error
	if _, err = c.Run(h.chart.Name()); errors.Is(err, driver.ErrReleaseNotFound) {
		h.logger.Info("Installing Helm Chart...")
		h.release, err = h.helmInstall(vals)
	} else {
		h.logger.Info("Upgrading Helm Chart...")
		h.release, err = h.helmUpgrade(vals)
	}
	if err != nil {
		return err
	}
	h.printRelease(h.release)
	return nil
}

// Verify equivalent to "helm test", it checks whether the release is correctly
// deployed by running chart tests and waiting for successful result.
func (h *Helm) Verify() error {
	if h.flags.DryRun {
		h.logger.Debug("Dry-run mode enabled, skipping verification")
		return nil
	}

	h.logger.Debug("Verifying the release...")
	c := action.NewReleaseTesting(h.actionCfg)
	c.Namespace = h.namespace

	_, err := c.Run(h.chart.Name())
	if err != nil {
		return err
	}
	h.logger.Info("Release verified!")
	return nil
}

// VisitReleaseResources collects the resources created by the Helm chart release.
func (h *Helm) VisitReleaseResources(
	ctx context.Context,
	m monitor.Interface,
) error {
	releasedResources, err := h.actionCfg.KubeClient.Build(
		bytes.NewBufferString(h.release.Manifest), true)
	if err != nil {
		return err
	}
	return releasedResources.Visit(func(r *resource.Info, err error) error {
		if err != nil {
			return err
		}
		return m.Collect(ctx, r)
	})
}

// NewHelm creates a new Helm instance, setting up the Helm action configuration
// to be used on subsequent interactions. The Helm instance is bound to a single
// Helm Chart.
func NewHelm(
	logger *slog.Logger,
	f *flags.Flags,
	kube *k8s.Kube,
	namespace string,
	chart *chart.Chart,
) (*Helm, error) {
	actionCfg := new(action.Configuration)
	getter := kube.RESTClientGetter(namespace)
	driver := os.Getenv("HELM_DRIVER")

	loggerFn := func(format string, v ...interface{}) {
		logger.WithGroup("helm-cli").Debug(fmt.Sprintf(format, v...))
	}
	err := actionCfg.Init(getter, namespace, driver, loggerFn)
	if err != nil {
		return nil, err
	}

	actionCfg.RegistryClient, err = registry.NewClient(
		registry.ClientOptDebug(true))
	if err != nil {
		return nil, err
	}

	return &Helm{
		logger: logger.With(
			"type", "helm",
			"chart", chart.Name(),
			"namespace", namespace,
		),
		flags:     f,
		chart:     chart,
		namespace: namespace,
		actionCfg: actionCfg,
	}, nil
}

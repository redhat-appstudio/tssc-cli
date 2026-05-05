package resolver

import (
	"fmt"
	"log/slog"
	"strconv"
	"strings"

	"github.com/redhat-appstudio/helmet/internal/annotations"
	"helm.sh/helm/v3/pkg/chart"
)

// Dependency represent a installer Dependency, which consists of a Helm chart
// instance, namespace and metadata. The relevant Helm chart metadata is read by
// helper methods.
type Dependency struct {
	chart     *chart.Chart // helm chart instance
	namespace string       // target namespace
	chartPath string       // chart directory relative to ChartFS (for per-chart values.yaml.tpl)
}

// Dependencies represents a slice of Dependency instances.
type Dependencies []Dependency

// LoggerWith decorates the logger with dependency information.
func (d *Dependency) LoggerWith(logger *slog.Logger) *slog.Logger {
	return logger.With(
		"dependency-name", d.Name(),
		"dependency-namespace", d.Namespace(),
		"dependency-chart-dir", d.ChartPath(),
	)
}

// Chart exposes the Helm chart instance.
func (d *Dependency) Chart() *chart.Chart {
	return d.chart
}

// Name returns the name of the Helm chart.
func (d *Dependency) Name() string {
	return d.chart.Name()
}

// Namespace returns the namespace.
func (d *Dependency) Namespace() string {
	return d.namespace
}

// ChartPath returns the filesystem path of the chart directory relative to ChartFS,
// used to resolve chart-local values.yaml.tpl.
func (d *Dependency) ChartPath() string {
	return d.chartPath
}

// SetNamespace sets the namespace for this dependency.
func (d *Dependency) SetNamespace(namespace string) {
	d.namespace = namespace
}

// getAnnotation retrieves a chart annotation value, returns empty for unknown
// annotation names.
func (d *Dependency) getAnnotation(annotation string) string {
	if v, exists := d.chart.Metadata.Annotations[annotation]; exists {
		return v
	}
	return ""
}

// DependsOn returns a slice of dependencies names from the chart's annotation.
func (d *Dependency) DependsOn() []string {
	dependsOn := d.getAnnotation(annotations.DependsOn)
	if dependsOn == "" {
		return nil
	}
	return commaSeparatedToSlice(dependsOn)
}

// Weight returns the weight of this dependency. If no weight is specified, zero
// is returned. The weight must be specified as an integer value.
func (d *Dependency) Weight() (int, error) {
	if v, exists := d.chart.Metadata.Annotations[annotations.Weight]; exists {
		w, err := strconv.Atoi(v)
		if err != nil {
			return -1, fmt.Errorf(
				"invalid value %q for annotation %q", v, annotations.Weight)
		}
		return w, nil
	}
	return 0, nil
}

// ProductName returns the product name from the chart annotations.
func (d *Dependency) ProductName() string {
	return d.getAnnotation(annotations.ProductName)
}

// UseProductNamespace returns the product namespace from the chart annotations.
func (d *Dependency) UseProductNamespace() string {
	return d.getAnnotation(annotations.UseProductNamespace)
}

// InstallReleaseInInstallerNamespace is true when the chart requests the Helm
// release to be installed in the installer namespace regardless of product namespace.
func (d *Dependency) InstallReleaseInInstallerNamespace() bool {
	v := strings.TrimSpace(strings.ToLower(
		d.getAnnotation(annotations.InstallReleaseInInstallerNamespace)))
	switch v {
	case "true", "1", "yes", "y":
		return true
	default:
		return false
	}
}

// IntegrationsProvided returns the integrations provided.
func (d *Dependency) IntegrationsProvided() []string {
	provided := d.getAnnotation(annotations.IntegrationsProvided)
	return commaSeparatedToSlice(provided)
}

// IntegrationsRequired returns the integrations required.
func (d *Dependency) IntegrationsRequired() string {
	return d.getAnnotation(annotations.IntegrationsRequired)
}

// BundleType returns the legacy bundle-type annotation (integration, product, dual).
func (d *Dependency) BundleType() string {
	return d.getAnnotation(annotations.BundleType)
}

// BundleSupport returns whether this chart may be deployed as an integration bundle
// and/or as a full product, based on bundle-types-supported (or legacy bundle-type).
func (d *Dependency) BundleSupport() (integration, product bool, err error) {
	return annotations.ParseBundleTypesSupported(
		d.getAnnotation(annotations.BundleTypesSupported),
		d.getAnnotation(annotations.BundleType),
	)
}

// SupportsIntegrationBundle is true when the chart may be listed under installer.integrations.
func (d *Dependency) SupportsIntegrationBundle() bool {
	i, _, err := d.BundleSupport()
	if err != nil {
		return false
	}
	return i
}

// SupportsProductBundle is true when the chart may be listed under products.
func (d *Dependency) SupportsProductBundle() bool {
	_, p, err := d.BundleSupport()
	if err != nil {
		return false
	}
	return p
}

// NewDependency creates a new Dependency for the Helm chart and initially using
// empty target namespace.
func NewDependency(hc *chart.Chart) *Dependency {
	return NewDependencyWithChartPath(hc, "")
}

// NewDependencyWithChartPath creates a Dependency that knows its chart directory
// on ChartFS for per-chart values templates.
func NewDependencyWithChartPath(hc *chart.Chart, chartPath string) *Dependency {
	return &Dependency{chart: hc, chartPath: chartPath}
}

// NewDependencyWithNamespace creates a new Dependency for the Helm chart and sets
// the target namespace.
func NewDependencyWithNamespace(hc *chart.Chart, ns string) *Dependency {
	return NewDependencyWithNamespaceAndChartPath(hc, ns, "")
}

// NewDependencyWithNamespaceAndChartPath creates a Dependency with namespace and
// chart directory path.
func NewDependencyWithNamespaceAndChartPath(hc *chart.Chart, ns, chartPath string) *Dependency {
	d := NewDependencyWithChartPath(hc, chartPath)
	d.SetNamespace(ns)
	return d
}

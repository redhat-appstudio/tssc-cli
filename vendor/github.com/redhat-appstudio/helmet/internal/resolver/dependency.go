package resolver

import (
	"fmt"
	"log/slog"
	"strconv"

	"github.com/redhat-appstudio/helmet/internal/annotations"
	"helm.sh/helm/v3/pkg/chart"
)

// Dependency represent a installer Dependency, which consists of a Helm chart
// instance, namespace and metadata. The relevant Helm chart metadata is read by
// helper methods.
type Dependency struct {
	chart     *chart.Chart // helm chart instance
	namespace string       // target namespace
}

// Dependencies represents a slice of Dependency instances.
type Dependencies []Dependency

// LoggerWith decorates the logger with dependency information.
func (d *Dependency) LoggerWith(logger *slog.Logger) *slog.Logger {
	return logger.With(
		"dependency-name", d.Name(),
		"dependency-namespace", d.Namespace(),
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

// IntegrationsProvided returns the integrations provided
func (d *Dependency) IntegrationsProvided() []string {
	provided := d.getAnnotation(annotations.IntegrationsProvided)
	return commaSeparatedToSlice(provided)
}

// IntegrationsRequired returns the integrations required.
func (d *Dependency) IntegrationsRequired() string {
	return d.getAnnotation(annotations.IntegrationsRequired)
}

// NewDependency creates a new Dependency for the Helm chart and initially using
// empty target namespace.
func NewDependency(hc *chart.Chart) *Dependency {
	return &Dependency{chart: hc}
}

// NewDependencyWithNamespace creates a new Dependency for the Helm chart and sets
// the target namespace.
func NewDependencyWithNamespace(hc *chart.Chart, ns string) *Dependency {
	d := NewDependency(hc)
	d.SetNamespace(ns)
	return d
}

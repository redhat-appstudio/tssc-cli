package resolver

import (
	"log/slog"
	"strings"

	"helm.sh/helm/v3/pkg/chart"
)

// Dependency represent a installer Dependency, which consists of a Helm chart
// instance, namespace and metadata. The relevant Helm chart metadata is read by
// helper methods.
type Dependency struct {
	chart     *chart.Chart // Helm chart instance
	namespace string       // Target namespace name
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

// DependsOn returns a slice of dependencies names from the chart's annotation.
func (d *Dependency) DependsOn() []string {
	dependsOn, exists := d.chart.Metadata.Annotations[DependsOnAnnotation]
	if !exists {
		return nil
	}
	dependsOn = strings.TrimSpace(dependsOn)
	if dependsOn == "" {
		return nil
	}
	parts := strings.Split(dependsOn, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if name := strings.TrimSpace(p); name != "" {
			out = append(out, name)
		}
	}
	return out
}

// ProductName returns the product name from the chart annotations.
func (d *Dependency) ProductName() string {
	name, exists := d.chart.Metadata.Annotations[ProductNameAnnotation]
	if exists {
		return name
	}
	return ""
}

// UseProductNamespace returns the product namespace from the chart annotations.
func (d *Dependency) UseProductNamespace() string {
	ns, exists := d.chart.Metadata.Annotations[UseProductNamespaceAnnotation]
	if exists {
		return ns
	}
	return ""
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

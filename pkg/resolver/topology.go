package resolver

import (
	"fmt"

	"github.com/redhat-appstudio/tssc-cli/pkg/config"
)

// Topology represents the dependency topology of charts. It is used to determine
// the order in which charts should be installed.
type Topology struct {
	dependencies config.Dependencies // dependency topology
}

// GetDependencies returns the list of dependencies.
func (t *Topology) GetDependencies() config.Dependencies {
	return t.dependencies
}

// GetDependencyForChart returns the dependency for a given chart name.
func (t *Topology) GetDependencyForChart(
	name string,
) (*config.Dependency, error) {
	for i := range t.dependencies {
		if t.dependencies[i].Chart.Name() == name {
			return &t.dependencies[i], nil
		}
	}
	return nil, fmt.Errorf("dependency not found for chart %s", name)
}

// Contains checks if a dependency Contains in the topology.
func (t *Topology) Contains(name string) bool {
	for _, d := range t.dependencies {
		if d.Chart.Name() == name {
			return true
		}
	}
	return false
}

// PrependBefore prepends a list of dependencies before a specific chart.
func (t *Topology) PrependBefore(name string, dependencies ...config.Dependency) {
	prefix := config.Dependencies{}
	for _, dependency := range dependencies {
		if !t.Contains(dependency.Chart.Name()) {
			prefix = append(prefix, dependency)
		}
	}
	if len(prefix) == 0 {
		return
	}

	// Find the index where the chart name exists.
	insertIndex := -1
	for i, d := range t.dependencies {
		if d.Chart.Name() == name {
			insertIndex = i
			break
		}
	}

	// Insert the prefix slice before the found chart name index. If the chart is
	// not found, prepend to the very beginning of the slice.
	if insertIndex != -1 {
		t.dependencies = append(
			t.dependencies[:insertIndex],
			append(prefix, t.dependencies[insertIndex:]...)...,
		)
	} else {
		t.dependencies = append(prefix, t.dependencies...)
	}
}

// AppendAfter inserts dependencies after a given chart name. If the chart does
// not exist, it appends to the end the slice.
func (t *Topology) AppendAfter(name string, dependencies ...config.Dependency) {
	suffix := config.Dependencies{}
	for _, dependency := range dependencies {
		if !t.Contains(dependency.Chart.Name()) {
			suffix = append(suffix, dependency)
		}
	}
	if len(suffix) == 0 {
		return
	}

	// Find the index where the chart name exists.
	insertIndex := -1
	for i, d := range t.dependencies {
		if d.Chart.Name() == name {
			insertIndex = i
			break
		}
	}

	// Insert the suffix slice after the found chart name index. If the chart is
	// not found, append it.
	if insertIndex != -1 {
		t.dependencies = append(
			t.dependencies[:insertIndex+1],
			append(suffix, t.dependencies[insertIndex+1:]...)...,
		)
	} else {
		t.dependencies = append(t.dependencies, suffix...)
	}
}

// Append adds a new dependency to the end of the topology.
func (t *Topology) Append(dependency config.Dependency) {
	if t.Contains(dependency.Chart.Name()) {
		return
	}
	t.dependencies = append(t.dependencies, dependency)
}

// NewTopology creates a new topology instance.
func NewTopology() *Topology {
	return &Topology{
		dependencies: config.Dependencies{},
	}
}

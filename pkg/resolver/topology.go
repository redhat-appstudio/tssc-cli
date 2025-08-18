package resolver

import (
	"fmt"
)

// Topology represents the dependency topology, determines the order in which
// charts (dependencies) will be installed.
type Topology struct {
	dependencies Dependencies // dependency topology
}

// Dependencies exposes the list of dependencies.
func (t *Topology) Dependencies() Dependencies {
	return t.dependencies
}

// GetDependency returns the dependency for a given dependency name.
func (t *Topology) GetDependency(name string) (*Dependency, error) {
	for i := range t.dependencies {
		if t.dependencies[i].Name() == name {
			return &t.dependencies[i], nil
		}
	}
	return nil, fmt.Errorf("dependency %q not found", name)
}

// Contains checks if a dependency Contains in the topology.
func (t *Topology) Contains(name string) bool {
	for _, d := range t.dependencies {
		if d.Name() == name {
			return true
		}
	}
	return false
}

// PrependBefore prepends a list of dependencies before a specific dependency.
func (t *Topology) PrependBefore(name string, dependencies ...Dependency) {
	prefix := Dependencies{}
	for _, dependency := range dependencies {
		if !t.Contains(dependency.Name()) {
			prefix = append(prefix, dependency)
		}
	}
	if len(prefix) == 0 {
		return
	}

	// Find the index where the dependency name exists.
	insertIndex := -1
	for i, d := range t.dependencies {
		if d.Name() == name {
			insertIndex = i
			break
		}
	}

	// Insert the prefix slice before the found dependency name index. If the
	// dependency is not found, prepend to the very beginning of the slice.
	if insertIndex != -1 {
		t.dependencies = append(
			t.dependencies[:insertIndex],
			append(prefix, t.dependencies[insertIndex:]...)...,
		)
	} else {
		t.dependencies = append(prefix, t.dependencies...)
	}
}

// AppendAfter inserts dependencies after a given dependency name. If the
// dependency does not exist, it appends to the end the slice.
func (t *Topology) AppendAfter(name string, dependencies ...Dependency) {
	suffix := Dependencies{}
	for _, d := range dependencies {
		if !t.Contains(d.Name()) {
			suffix = append(suffix, d)
		}
	}
	if len(suffix) == 0 {
		return
	}

	// Find the index where the dependency name exists.
	insertIndex := -1
	for i, d := range t.dependencies {
		if d.Name() == name {
			insertIndex = i
			break
		}
	}
	// Insert the suffix slice after the found dependency name index. If the
	// dependency is not found, append it.
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
func (t *Topology) Append(d Dependency) {
	if t.Contains(d.Name()) {
		return
	}
	t.dependencies = append(t.dependencies, d)
}

// NewTopology creates a new topology instance.
func NewTopology() *Topology {
	return &Topology{
		dependencies: Dependencies{},
	}
}

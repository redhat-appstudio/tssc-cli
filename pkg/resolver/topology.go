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

// Walk traverses the topology and calls the informed function for each
// dependency.
func (t *Topology) Walk(fn DependencyWalkFn) error {
	for i := range t.dependencies {
		if err := fn(t.dependencies[i].Name(), t.dependencies[i]); err != nil {
			return err
		}
	}
	return nil
}

// dependencyIndex given the dependency name, find the its index in the topology,
// or returns -1 if not found.
func (t *Topology) dependencyIndex(name string) int {
	for i, d := range t.dependencies {
		if d.Name() == name {
			return i
		}
	}
	return -1
}

// except returns a list of dependencies that are not in the topology.
func (t *Topology) except(dependencies ...Dependency) Dependencies {
	except := Dependencies{}
	for _, dependency := range dependencies {
		if !t.Contains(dependency.Name()) {
			except = append(except, dependency)
		}
	}
	return except
}

// PrependBefore prepends a list of dependencies before a specific dependency,
// taking the weight into account.
func (t *Topology) PrependBefore(name string, dependencies ...Dependency) {
	except := t.except(dependencies...)
	if len(except) == 0 {
		return
	}

	// Find the index where the dependency name exists.
	dependencyIndex := t.dependencyIndex(name)

	// The dependency is not found, prepend to the very beginning of the slice.
	if dependencyIndex == -1 {
		t.dependencies = append(except, t.dependencies...)
		return
	}

	insertIndex := dependencyIndex

	// Calculate the insert index based on weights.
	for _, dep := range except {
		pos := insertIndex
		currentWeight, _ := dep.Weight()

		for pos > 0 {
			prevWeight, _ := t.dependencies[pos-1].Weight()
			if currentWeight >= prevWeight {
				break
			}
			pos--
		}

		t.dependencies = append(
			t.dependencies[:pos],
			append([]Dependency{dep}, t.dependencies[pos:]...)...,
		)

		dependencyIndex++
		insertIndex = dependencyIndex
	}
}

// AppendAfter inserts dependencies after a given dependency name. If the
// dependency does not exist, it appends to the end the slice.
func (t *Topology) AppendAfter(name string, dependencies ...Dependency) {
	except := t.except(dependencies...)
	if len(except) == 0 {
		return
	}

	// Find the index where the dependency name exists.
	dependencyIndex := t.dependencyIndex(name)

	if dependencyIndex == -1 {
		t.dependencies = append(t.dependencies, except...)
		return
	}

	// The insert index starts right next to the dependency name.
	insertIndex := dependencyIndex + 1

	// Calculate the insert index based on weights.
	for _, dep := range except {
		pos := insertIndex
		currentWeight, _ := dep.Weight()

		for pos < len(t.dependencies) {
			nextWeight, _ := t.dependencies[pos].Weight()
			if currentWeight <= nextWeight {
				break
			}
			pos++
		}

		t.dependencies = append(
			t.dependencies[:pos],
			append([]Dependency{dep}, t.dependencies[pos:]...)...,
		)
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

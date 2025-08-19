package resolver

import (
	"fmt"
	"io"
	"strings"
	"text/tabwriter"

	"github.com/redhat-appstudio/tssc-cli/pkg/config"
)

// Resolver represents the actor that resolves dependencies between charts.
type Resolver struct {
	cfg        *config.Config // installer configuration
	collection *Collection    // collection of charts
	topology   *Topology      // topology of dependencies
}

// ErrCircularDependency reports a circular dependency.
var ErrCircularDependency = fmt.Errorf("circular dependency detected")

// ErrMissingDependency reports an unmet dependency.
var ErrMissingDependency = fmt.Errorf("unmet dependency detected")

// setDependencyNamespace sets the desired namespace on the informed dependency.
// By default, charts are deployed on the same namespace than the installer, while
// product assossiated dependencies will use the namespace configured for it.
func (r *Resolver) setDependencyNamespace(d *Dependency) error {
	var product string
	// Check if the Helm chart should use the product namespace.
	if p := d.UseProductNamespace(); p != "" {
		product = p
	}
	// Check if the Helm chart is associated with a product, which takes
	// precedence over the "use-product-namespace" annotation.
	if p := d.ProductName(); p != "" {
		product = p
	}

	// Choosing the namespace for the dependency, a product chart will use what's
	// defined for it, while regular charts will use the installer's namespace.
	var namespace string
	if product == "" {
		namespace = r.cfg.Installer.Namespace
	} else {
		spec, err := r.cfg.GetProduct(product)
		if err != nil {
			return err
		}
		namespace = *spec.Namespace
	}
	d.SetNamespace(namespace)
	return nil
}

// dependsOn checks if the chart has dependencies and resolves them. The
// dependencies are prepended to the parent chart and when more dependencies are
// found, they are also resolved.
func (r *Resolver) dependsOn(
	parent string, // partent chart name
	d *Dependency, // dependency instance
	visited map[string]bool, // visited charts
) error {
	// Ensure the chart is not visited again, to prevent circular dependencies.
	dependencyName := d.Name()
	if visited[dependencyName] {
		return fmt.Errorf("%w: a %q dependency requires %q",
			ErrCircularDependency, dependencyName, dependencyName)
	}
	visited[dependencyName] = true
	defer delete(visited, dependencyName)

	for _, dependsOn := range d.DependsOn() {
		// Picking up the dependency from the collection by name.
		dependsOnDep, err := r.collection.Get(dependsOn)
		if err != nil {
			return err
		}
		// Skiping when the next dependency is associated with a disabled product.
		if product := dependsOnDep.ProductName(); product != "" {
			productSpec, err := r.cfg.GetProduct(product)
			if err != nil {
				return err
			}
			if !productSpec.Enabled {
				continue
			}
		}
		// Setting the correct namespace in the dependency.
		if err := r.setDependencyNamespace(dependsOnDep); err != nil {
			return err
		}
		// Adding the Helm chart to the topology before the parent chart. The
		// namespace is the installer's default.
		r.topology.PrependBefore(parent, *dependsOnDep)
		// Recursively resolving the dependencies.
		if err = r.dependsOn(dependsOn, dependsOnDep, visited); err != nil {
			return err
		}
	}
	return nil
}

// resolveEnabledProducts resolves the dependencies of enabled products.
func (r *Resolver) resolveEnabledProducts() error {
	for _, product := range r.cfg.GetEnabledProducts() {
		d, err := r.collection.GetProductDependency(product.Name)
		if err != nil {
			return err
		}
		// Products uses the namespace specified in the configuration.
		d.SetNamespace(*product.Namespace)
		// Product charts are added to the topology before required charts.
		r.topology.Append(*d)
		// Recursively resolving the dependencies, added before this chart.
		if err = r.dependsOn(d.Name(), d, map[string]bool{}); err != nil {
			return err
		}
	}
	return nil
}

// resolveDependencies final inspection of the Helm charts in the Collection to
// ensure all dependencies are met. It walks the charts in the Collection, and for
// each entry verifies it it depends on any chart in the Topology.
func (r *Resolver) resolveDependencies() error {
	return r.collection.Walk(func(name string, d Dependency) error {
		// Skip dependencies that are associated with a product. These have
		// already been added to the topology.
		if product := d.ProductName(); product != "" {
			return nil
		}
		// Collecting the last dependency name that is required by the current
		// chart (dependency), if any.
		requiredBy := ""
		for _, dependsOn := range d.DependsOn() {
			// Ensure the required dependency is in the topology, when not in the
			// topology it is skipped.
			if !r.topology.Contains(dependsOn) {
				continue
			}
			// Ensures the required dependency is in the collection.
			if _, err := r.collection.Get(dependsOn); err != nil {
				return fmt.Errorf(
					"%w: dependency %s not found for chart %s",
					ErrMissingDependency,
					dependsOn,
					name,
				)
			}
			requiredBy = dependsOn
		}
		// If it's not required by any other dependency, skip it.
		if requiredBy == "" {
			return nil
		}
		// Setting the desired namespace in the dependency.
		if err := r.setDependencyNamespace(&d); err != nil {
			return err
		}
		// Append the current dependency after the last one in the collection that
		// requires it.
		r.topology.AppendAfter(requiredBy, d)
		// Recursively resolve dependencies.
		return r.dependsOn(name, &d, map[string]bool{})
	})
}

// Resolve resolves the all dependencies in the collection to create the topology.
func (r *Resolver) Resolve() error {
	if err := r.resolveEnabledProducts(); err != nil {
		return err
	}
	return r.resolveDependencies()
}

// Print prints the resolved topology to the writer formatted as a table.
func (r *Resolver) Print(w io.Writer) {
	table := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	row := func(a ...any) {
		fmt.Fprintf(table, "%s\t%s\t%s\t%s\t%s\n", a...)
	}
	row("Index", "Dependency", "Namespace", "Product", "Depends-On")
	for i, d := range r.topology.Dependencies() {
		dependsOn := d.DependsOn()
		row(
			fmt.Sprintf("%2d", i+1),
			d.Name(),
			d.Namespace(),
			d.ProductName(),
			strings.Join(dependsOn, ", "),
		)
	}
	table.Flush()
}

// NewResolver instantiates a new Resolver. It takes the configuration, collection
// and topology as parameters.
func NewResolver(cfg *config.Config, c *Collection, t *Topology) *Resolver {
	return &Resolver{
		cfg:        cfg,
		collection: c,
		topology:   t,
	}
}

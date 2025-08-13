package resolver

import (
	"fmt"
	"io"
	"strings"
	"text/tabwriter"

	"github.com/redhat-appstudio/tssc-cli/pkg/config"

	"helm.sh/helm/v3/pkg/chart"
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

// dependency returns the dependency for a chart. If the chart is associated with
// a product, the dependency use the product namespace. If the chart contains a
// "use-product-namespace" annotation, the dependency use the product namespace.
// If no product is associated with the chart, the dependency use the installer
// namespace.
func (r *Resolver) dependency(hc *chart.Chart) (*config.Dependency, error) {
	var product string
	// Check if the Helm chart should use the product namespace.
	if p := r.collection.UseProductNamespace(hc); p != "" {
		product = p
	}
	// Check if the Helm chart is associated with a product, which takes
	// precedence the "use-product-namespace" annotation.
	if p := r.collection.ProductName(hc); p != "" {
		product = p
	}
	// When no product is found, use the installer namespace.
	if product == "" {
		return config.NewDependency(hc, r.cfg.Installer.Namespace), nil
	}
	// Given a product name is found, using the product namespace.
	spec, err := r.cfg.GetProduct(product)
	if err != nil {
		return nil, err
	}
	return config.NewDependency(hc, *spec.Namespace), nil
}

// dependsOn checks if the chart has dependencies and resolves them. The
// dependencies are prepended to the parent chart and when more dependencies are
// found, they are also resolved.
func (r *Resolver) dependsOn(
	parent string, // partent chart name
	hc *chart.Chart, // helm chart instance
	visited map[string]bool, // visited charts
) error {
	// Ensure the chart is not visited again, to prevent circular dependencies.
	chartName := hc.Name()
	if visited[chartName] {
		return fmt.Errorf("%w: a %q dependency requires %q",
			ErrCircularDependency, chartName, chartName)
	}
	visited[chartName] = true
	defer delete(visited, chartName)

	for _, dependsOn := range r.collection.DependsOn(hc) {
		dependsOnHC, err := r.collection.Get(dependsOn)
		if err != nil {
			return err
		}
		// Skiping when the Helm chart is associated with a disabled product.
		if product := r.collection.ProductName(dependsOnHC); product != "" {
			productSpec, err := r.cfg.GetProduct(product)
			if err != nil {
				return err
			}
			if !productSpec.Enabled {
				continue
			}
		}
		dep, err := r.dependency(dependsOnHC)
		if err != nil {
			return err
		}
		// Adding the Helm chart to the topology before the parent chart. The
		// namespace is the installer's default.
		r.topology.PrependBefore(parent, *dep)
		// Recursively resolving the dependencies.
		if err = r.dependsOn(dependsOn, dependsOnHC, visited); err != nil {
			return err
		}
	}
	return nil
}

// resolveEnabledProducts resolves the dependencies of enabled products.
func (r *Resolver) resolveEnabledProducts() error {
	for _, product := range r.cfg.GetEnabledProducts() {
		hc, err := r.collection.GetProductChart(product.Name)
		if err != nil {
			return err
		}
		// Product charts are added to the topology before required charts.
		r.topology.Append(*config.NewDependency(hc, *product.Namespace))
		// Recursively resolving the dependencies, added before this chart.
		if err = r.dependsOn(hc.Name(), hc, map[string]bool{}); err != nil {
			return err
		}
	}
	return nil
}

// resolveDependencies final inspection of the Helm charts in the Collection to
// ensure all dependencies are met. It walks the charts in the Collection, and for
// each entry verifies it it depends on any chart in the Topology.
func (r *Resolver) resolveDependencies() error {
	return r.collection.Walk(func(name string, hc chart.Chart) error {
		// Skip charts that are associated with a product. These charts are
		// already added to the topology.
		if productName := r.collection.ProductName(&hc); productName != "" {
			return nil
		}
		// Collecting the last Helm chart name that is required by the current
		// chart, if any.
		requiredBy := ""
		for _, dependsOn := range r.collection.DependsOn(&hc) {
			// Ensure the required chart is in the topology, when not in the
			// topology it is skipped.
			if !r.topology.Contains(dependsOn) {
				continue
			}
			// Ensures the if the required chart is in the collection.
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
		// If it's not required by any other chart, skip it.
		if requiredBy == "" {
			return nil
		}
		dep, err := r.dependency(&hc)
		if err != nil {
			return err
		}
		// Append the current chart after the last chart in the collection that
		// required it.
		r.topology.AppendAfter(requiredBy, *dep)
		// Recursively resolve dependencies for the current chart.
		return r.dependsOn(name, &hc, map[string]bool{})
	})
}

// Resolve resolves the dependencies of the charts in the collection generating
// the installer topology.
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
	row("Index", "Chart", "Namespace", "Product", "Depends-On")
	for i, d := range r.topology.GetDependencies() {
		dependsOn := r.collection.DependsOn(d.Chart)
		row(
			fmt.Sprintf("%2d", i+1),
			d.Chart.Name(),
			d.Namespace,
			r.collection.ProductName(d.Chart),
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

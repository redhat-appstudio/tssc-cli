package resolver

import (
	"errors"
	"fmt"
	"slices"
	"strings"

	"helm.sh/helm/v3/pkg/chart"
)

// Collection represents a collection of charts the resolver can use. The
// collection is concise, all charts and product names are unique.
type Collection struct {
	charts map[string]*chart.Chart // chart instances by name
}

// CollectionWalkFn is a function that is called for each chart in the collection,
// the chart name and instance are passed to it.
type CollectionWalkFn func(string, chart.Chart) error

var (
	// ErrInvalidCollection the collection is invalid.
	ErrInvalidCollection error = errors.New("invalid collection")
	// ErrChartNotFound the chart is not found in the collection.l
	ErrChartNotFound error = errors.New("chart not found")
)

// Get returns the chart with the given name.
func (c *Collection) Get(name string) (*chart.Chart, error) {
	hc, exists := c.charts[name]
	if !exists {
		return nil, fmt.Errorf("%s: %s", ErrChartNotFound, name)
	}
	return hc, nil
}

// Walk iterates over all charts in the collection and calls the provided function
// for each entry.
func (c *Collection) Walk(fn CollectionWalkFn) error {
	names := make([]string, 0, len(c.charts))
	for name := range c.charts {
		names = append(names, name)
	}
	slices.Sort(names)

	for _, name := range names {
		if err := fn(name, *c.charts[name]); err != nil {
			return err
		}
	}
	return nil
}

// GetProductChart returns the first Helm chart that is associated with the
// informed product. Returns error when no chart is found.
func (c *Collection) GetProductChart(product string) (*chart.Chart, error) {
	var productHelmChart *chart.Chart
	_ = c.Walk(func(_ string, hc chart.Chart) error {
		// Already found the product chart, no need to continue.
		if productHelmChart != nil {
			return nil
		}
		// Check if the Helm chart is associated with the product.
		if name := c.ProductName(&hc); name != "" && name == product {
			productHelmChart = &hc
		}
		return nil
	})
	if productHelmChart == nil {
		return nil, fmt.Errorf("%w: for product %s", ErrChartNotFound, product)
	}
	return productHelmChart, nil
}

// DependsOn returns the list of chart names this chart depends on.
func (c *Collection) DependsOn(hc *chart.Chart) []string {
	dependsOn, exists := hc.Metadata.Annotations[DependsOnAnnotation]
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

// ProductName returns the product name for the given chart.
func (c *Collection) ProductName(hc *chart.Chart) string {
	if name, exists := hc.Metadata.Annotations[ProductNameAnnotation]; exists {
		return name
	}
	return ""
}

// UseProductNamespace returns the product namespace for the given chart.
func (c *Collection) UseProductNamespace(hc *chart.Chart) string {
	ns, exists := hc.Metadata.Annotations[UseProductNamespaceAnnotation]
	if exists {
		return ns
	}
	return ""
}

// populate initializes the collection with the provided charts. It checks for
// duplicate charts and product names, returning error.
func (c *Collection) populate(charts []chart.Chart) error {
	productNames := []string{}
	for _, hc := range charts {
		// Charts must have unique names.
		if _, err := c.Get(hc.Name()); err == nil {
			return fmt.Errorf(
				"%w: duplicate chart: %s",
				ErrInvalidCollection,
				hc.Name(),
			)
		}
		// Product names must be unique.
		if name := c.ProductName(&hc); name != "" {
			if slices.Contains(productNames, name) {
				return fmt.Errorf(
					"%w: duplicate product name: %s",
					ErrInvalidCollection,
					name,
				)
			}
			// Caching product names.
			productNames = append(productNames, name)
		}
		// Insert chart into collection.
		c.charts[hc.Name()] = &hc
	}
	return nil
}

// NewCollection creates a new Collection from the given charts. It returns an
// error if there are duplicate charts.
func NewCollection(charts []chart.Chart) (*Collection, error) {
	c := &Collection{charts: map[string]*chart.Chart{}}
	if err := c.populate(charts); err != nil {
		return nil, err
	}
	return c, nil
}

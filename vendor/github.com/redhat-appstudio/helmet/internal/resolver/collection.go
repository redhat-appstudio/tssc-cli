package resolver

import (
	"errors"
	"fmt"
	"slices"

	"github.com/redhat-appstudio/helmet/api"
	"helm.sh/helm/v3/pkg/chart"
)

// Collection represents a collection of dependencies the Resolver can utilize.
// The collection is concise, all dependencies and product names must be unique.
type Collection struct {
	dependencies map[string]*Dependency // dependencies by name
}

// DependencyWalkFn is a function that is called for each dependency in the
// collection, the dependency name and instance are passed to it.
type DependencyWalkFn func(string, Dependency) error

var (
	// ErrInvalidCollection the collection is invalid.
	ErrInvalidCollection error = errors.New("invalid collection")
	// ErrDependencyNotFound the dependency is not found in the collection.
	ErrDependencyNotFound error = errors.New("dependency not found")
)

// Get returns the dependency with the given name.
func (c *Collection) Get(name string) (*Dependency, error) {
	d, exists := c.dependencies[name]
	if !exists {
		return nil, fmt.Errorf("%s: %s", ErrDependencyNotFound, name)
	}
	return d, nil
}

// Walk iterates over all dependencies in the collection and calls the provided
// function for each entry.
func (c *Collection) Walk(fn DependencyWalkFn) error {
	names := make([]string, 0, len(c.dependencies))
	for name := range c.dependencies {
		names = append(names, name)
	}
	slices.Sort(names)

	for _, name := range names {
		if err := fn(name, *c.dependencies[name]); err != nil {
			return err
		}
	}
	return nil
}

// GetProductDependency returns the dependency associated with the informed
// product. Returns error when no dependency is found.
func (c *Collection) GetProductDependency(product string) (*Dependency, error) {
	var productDependency *Dependency
	_ = c.Walk(func(_ string, d Dependency) error {
		// Already found the product dependency, no need to continue.
		if productDependency != nil {
			return nil
		}
		// Check if the dependency is associated with the product.
		if name := d.ProductName(); name != "" && name == product {
			productDependency = &d
		}
		return nil
	})
	if productDependency == nil {
		return nil, fmt.Errorf("%w: for product %s",
			ErrDependencyNotFound, product)
	}
	return productDependency, nil
}

// GetProductNameForIntegration searches and returns the product name by integration name.
// It goes though all charts and search for annotation "integrations-provided"
// If it matches integration name, then returns product name, which is from
// annotation "product-name"
func (c *Collection) GetProductNameForIntegration(integrationName string) string {
	var productName string
	_ = c.Walk(func(_ string, d Dependency) error {
		if productName != "" {
			return nil // stop walking
		}

		if d.IntegrationsProvided() != nil &&
			slices.Contains(d.IntegrationsProvided(), integrationName) {
			productName = d.ProductName()
		}
		return nil
	})

	return productName
}

// NewCollection creates a new Collection from the given charts. It returns an
// error if there are duplicate charts and product names.
func NewCollection(appCtx *api.AppContext, charts []chart.Chart) (*Collection, error) {
	c := &Collection{dependencies: map[string]*Dependency{}}
	// Stores the product names found in the slice of Helm charts.
	productNames := []string{}
	// Populating the collection with dependencies.
	for _, hc := range charts {
		d := NewDependency(&hc)
		// Asserting the weight annotation is a valid integer.
		if _, err := d.Weight(); err != nil {
			return nil, fmt.Errorf("%w:  %s", ErrInvalidCollection, err)
		}
		// Dependencies in the collection must have unique names.
		if _, err := c.Get(d.Name()); err == nil {
			return nil, fmt.Errorf("%w: duplicate chart: %s",
				ErrInvalidCollection, d.Name(),
			)
		}
		// Product names must be unique.
		if name := d.ProductName(); name != "" {
			if slices.Contains(productNames, name) {
				return nil, fmt.Errorf("%w: duplicate product name: %s",
					ErrInvalidCollection, name)
			}
			// Caching product names.
			productNames = append(productNames, name)
		}
		// Insert the dependency into the collection.
		c.dependencies[d.Name()] = d
	}
	return c, nil
}

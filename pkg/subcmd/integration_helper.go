package subcmd

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"github.com/redhat-appstudio/tssc-cli/pkg/chartfs"
	"github.com/redhat-appstudio/tssc-cli/pkg/config"
	"github.com/redhat-appstudio/tssc-cli/pkg/k8s"
	"github.com/redhat-appstudio/tssc-cli/pkg/resolver"
)

// productName searches and returns the product name by integration name.
// It goes though all charts and search for annotation "integrations-provided"
// If it matches integration name, then returns product name, which is from
// annotation "product-name"
func productName(name string) (string, error) {
	cfs, err := chartfs.NewChartFSForCWD()
	if err != nil {
		return "", err
	}

	charts, err := cfs.GetAllCharts()
	if err != nil {
		return "", err
	}

	for _, hc := range charts {
		d := resolver.NewDependency(&hc)
		if slices.Contains(d.IntegrationsProvided(), name) {
			return d.ProductName(), nil
		}
	}

	return "", fmt.Errorf("failed to get product name from integration: %s", name)
}

// disableProduct disables a product by its name in the configuration.
// It searches for the product and updates its 'Enabled' field accordingly.
func disableProduct(
	name string,
	ctx context.Context,
	cfg *config.Config,
	kube *k8s.Kube,
) error {
	parts := strings.Split(name, "-")
	if len(parts) < 2 {
		return fmt.Errorf("invalid name format: %s", name)
	}

	pName, err := productName(parts[1])
	if err != nil {
		return err
	}

	spec, err := cfg.GetProduct(pName)
	if err != nil {
		return err
	}

	spec.Enabled = false
	if err := cfg.SetProduct(pName, *spec); err != nil {
		return err
	}

	return config.NewConfigMapManager(kube).Update(ctx, cfg)
}

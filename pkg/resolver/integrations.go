package resolver

import (
	"context"
	"errors"
	"fmt"

	"github.com/redhat-appstudio/tssc-cli/pkg/config"
	"github.com/redhat-appstudio/tssc-cli/pkg/integrations"
)

// Integrations represents the actor which inspects the integrations provided and
// required by each Helm chart (dependency) in the Topology.
type Integrations struct {
	configured map[string]bool // integration state machine
	cel        *CEL            // CEL environment
}

var (
	// ErrUnknownIntegration the integration name is not supported, unknown.
	ErrUnknownIntegration = errors.New("unknown integration")
	// ErrConfiguredIntegration the integration is already configured in the
	// cluster, either by the user or another dependency.
	ErrConfiguredIntegration = errors.New("integration is already configured")
)

// Inspect loops the Topology and evaluates the integrations required by each
// dependency, as well integrations provided by them. The inspection keeps the
// state of the integrations configured in the cluster.
func (i *Integrations) Inspect(t *Topology) error {
	return t.Walk(func(chartName string, d Dependency) error {
		// Inspecting the integrations required by the dependency, the "required"
		// annotation is a CEL expression describing which integration it depends
		// on. If the expression evaluates to false, the integration is not
		// configured in the cluster, and it's not provided by any other
		// dependency (chart) in the Topology.
		if required := d.IntegrationsRequired(); required != "" {
			if err := i.cel.Evaluate(i.configured, required); err != nil {
				return fmt.Errorf("%w: in %q dependency using expression %q",
					err, chartName, required)
			}
		}
		// Inspecting the integrations provided by the Helm chart (dependency). It
		// must provide a integration name supported by this project, and must not
		// overwrite configured integrations.
		for _, provided := range d.IntegrationsProvided() {
			configured, exists := i.configured[provided]
			// Asserting that the integration is provided by this project.
			if !exists {
				return fmt.Errorf("%w: %q in %q dependency (%q product)",
					ErrUnknownIntegration, provided, chartName, d.ProductName())
			}
			// Asserting the integration is not configured yet.
			if configured {
				return fmt.Errorf(
					"%w: %q can't be overwritten by the %q (%q product)",
					ErrConfiguredIntegration,
					provided,
					chartName,
					d.ProductName(),
				)
			}
			// Marking the integration as configured, this dependency is
			// responsible for creating the integration secret accordingly.
			i.configured[provided] = true
		}
		return nil
	})
}

// NewIntegrations creates a new Integrations instance. It populates the a map
// with the integrations that are currently configured in the cluster, marking the
// others as missing.
func NewIntegrations(
	ctx context.Context,
	cfg *config.Config,
	manager *integrations.Manager,
) (*Integrations, error) {
	i := &Integrations{configured: map[string]bool{}}

	// Populating the integration names configured in the cluster, representing
	// actual Kubernetes integration secrets existing in the cluster.
	configuredIntegrations, err := manager.ConfiguredIntegrations(ctx, cfg)
	if err != nil {
		return nil, err
	}
	// When the integration exists, it marks the integration name as true, so it's
	// configured in the cluster.
	for _, name := range configuredIntegrations {
		i.configured[name] = true
	}
	// Going through all valid integration names, by default when not registered
	// the integration name is marked as false, as in not configured in the
	// cluster.
	for _, name := range manager.IntegrationNames() {
		if _, exists := i.configured[name]; !exists {
			i.configured[name] = false
		}
	}
	// Bootstrapping the CEL environment with all known integration names.
	if i.cel, err = NewCEL(manager.IntegrationNames()...); err != nil {
		return nil, err
	}
	return i, nil
}

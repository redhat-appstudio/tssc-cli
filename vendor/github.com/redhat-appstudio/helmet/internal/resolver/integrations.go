package resolver

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/redhat-appstudio/helmet/internal/config"
	"github.com/redhat-appstudio/helmet/internal/integrations"
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
	// ErrPrerequisiteIntegration dependency prerequisite integration(s) missing.
	ErrPrerequisiteIntegration = errors.New(
		"dependency prerequisite integration(s) missing")
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
				switch {
				case errors.Is(err, ErrMissingIntegrations):
					return fmt.Errorf(
						`%w:

The dependency %q requires specific set of cluster integrations,
defined by the following CEL expression:

	%q

This expression was evaluated against the cluster's configured integrations, and
the evaluation failed. The following integration names are present in the
expression but not configured in the cluster:

	%q`,
						ErrPrerequisiteIntegration,
						chartName,
						required,
						strings.TrimPrefix(
							err.Error(),
							fmt.Sprintf("%s: ", ErrMissingIntegrations),
						),
					)
				case errors.Is(err, ErrInvalidExpression):
					return fmt.Errorf(
						`%w:

The dependency %q defines an invalid CEL expression for required
cluster integrations:

	%q

The CEL evaluation failed with the following error:

	%q`,
						ErrInvalidExpression, chartName, required, err.Error(),
					)
				default:
					return fmt.Errorf(
						`%w:

The dependency %q requires specific set of cluster integrations,
defined by the following CEL expression:

	%q

An unexpected error occurred during CEL evaluation:

	%q`,
						ErrPrerequisiteIntegration,
						chartName,
						required,
						err.Error(),
					)
				}
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
			if configured {
				// If the integration is already configured (either by user or
				// previous run) we skip marking it again to ensure idempotency.
				continue
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

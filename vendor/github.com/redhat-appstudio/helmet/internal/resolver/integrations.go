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

// Inspect walks the Topology in two passes to evaluate integrations provided and
// required by each dependency. The two-pass approach makes validation
// order-independent: all provisions are collected first, then all requirements
// are evaluated against the complete state.
func (i *Integrations) Inspect(t *Topology) error {
	// Pass 1: collect all integrations provided by charts in the topology.
	// This marks each provided integration as configured before any
	// requirements are evaluated, eliminating order-dependency.
	if err := t.Walk(func(chartName string, d Dependency) error {
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
	}); err != nil {
		return err
	}

	// Pass 2: validate all integrations required by charts in the topology.
	// At this point the configured map contains both cluster-state entries and
	// all provisions declared by charts, so CEL evaluation is independent of
	// topology ordering.
	return t.Walk(func(chartName string, d Dependency) error {
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

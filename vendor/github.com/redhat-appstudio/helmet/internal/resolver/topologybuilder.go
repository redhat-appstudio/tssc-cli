package resolver

import (
	"context"
	"log/slog"

	"github.com/redhat-appstudio/helmet/api"
	"github.com/redhat-appstudio/helmet/internal/chartfs"
	"github.com/redhat-appstudio/helmet/internal/config"
	"github.com/redhat-appstudio/helmet/internal/integrations"
)

// TopologyBuilder represents the complete workflow to generate a consolidated
// Topology, with dependencies and integrations verified.
type TopologyBuilder struct {
	logger              *slog.Logger          // application logger
	collection          *Collection           // charts collection
	integrationsManager *integrations.Manager // integrations manager
}

// GetCollection exposes the collection instance.
func (t *TopologyBuilder) GetCollection() *Collection {
	return t.collection
}

// Build inspects the dependencies, based on the cluster configuration, inspects
// the integrations and generates a consolidated Topology
func (t *TopologyBuilder) Build(
	ctx context.Context,
	cfg *config.Config,
) (*Topology, error) {
	topology := NewTopology()
	r := NewResolver(cfg, t.collection, topology)

	// Inspecting all charts, dependencies, to organize the topology, which is the
	// sequence of dependencies deployment.
	t.logger.Debug("Resolving the topology dependencies...")
	err := r.Resolve()
	if err != nil {
		return nil, err
	}
	// Given the Topology is created, now the integrations are verified to ensure
	// all required integrations secrets are configured.
	t.logger.Debug("Inspecting integrations...")
	i, err := NewIntegrations(ctx, cfg, t.integrationsManager)
	if err != nil {
		return nil, err
	}
	t.logger.Debug("Asserting all required integrations are configured...")
	if err = i.Inspect(topology); err != nil {
		return nil, err
	}
	return topology, nil
}

// NewTopologyBuilder creates a new TopologyBuilder instance.
func NewTopologyBuilder(
	appCtx *api.AppContext,
	logger *slog.Logger,
	cfs *chartfs.ChartFS,
	integrationsManager *integrations.Manager,
) (*TopologyBuilder, error) {
	t := &TopologyBuilder{
		logger:              logger,
		integrationsManager: integrationsManager,
	}
	// Reading all charts from the informed filesystem.
	charts, err := cfs.GetAllCharts()
	if err != nil {
		return nil, err
	}
	// Creating a collection with the charts found.
	if t.collection, err = NewCollection(appCtx, charts); err != nil {
		return nil, err
	}
	return t, nil
}

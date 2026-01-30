package integrations

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/redhat-appstudio/helmet/api"
	"github.com/redhat-appstudio/helmet/internal/config"
	"github.com/redhat-appstudio/helmet/internal/integration"
	"github.com/redhat-appstudio/helmet/internal/k8s"
)

// IntegrationName name of a integration.
type IntegrationName string

// Manager represents the actor responsible for all integrations.
// It centralizes the management of integration instances, keeping a consistent
// set of integration names.
type Manager struct {
	integrations map[IntegrationName]*integration.Integration // integrations
	modules      map[IntegrationName]api.IntegrationModule    // modules
}

const (
	ACS                   IntegrationName = "acs"
	Artifactory           IntegrationName = "artifactory"
	Azure                 IntegrationName = "azure"
	BitBucket             IntegrationName = "bitbucket"
	GitHub                IntegrationName = "github"
	GitLab                IntegrationName = "gitlab"
	Jenkins               IntegrationName = "jenkins"
	Nexus                 IntegrationName = "nexus"
	Quay                  IntegrationName = "quay"
	TrustedArtifactSigner IntegrationName = "tas"
	Trustification        IntegrationName = "trustification"
)

// Integration returns the integration instance by name.
func (m *Manager) Integration(name IntegrationName) *integration.Integration {
	i, exists := m.integrations[name]
	if !exists {
		panic(fmt.Sprintf("integration instance is not found: %q", name))
	}
	return i
}

// IntegrationNames returns a list of all integration names.
func (m *Manager) IntegrationNames() []string {
	names := []string{}
	for name := range m.integrations {
		names = append(names, string(name))
	}
	return names
}

// GetModules returns the list of registered integration modules.
func (m *Manager) GetModules() []api.IntegrationModule {
	modules := []api.IntegrationModule{}
	for _, mod := range m.modules {
		modules = append(modules, mod)
	}
	return modules
}

// ConfiguredIntegrations returns a slice of integration names configured in the
// cluster, it uses the "Exists" method in the integration instance to assert it's
// secret is present in the cluster.
func (m *Manager) ConfiguredIntegrations(
	ctx context.Context,
	cfg *config.Config,
) ([]string, error) {
	configured := []string{}
	for name, i := range m.integrations {
		exists, err := i.Exists(ctx, cfg)
		if err != nil {
			return nil, err
		}
		if exists {
			configured = append(configured, string(name))
		}
	}
	return configured, nil
}

// Register adds a integration instance to the manager.
func (m *Manager) Register(mod api.IntegrationModule, i *integration.Integration) {
	name := IntegrationName(mod.Name)
	m.integrations[name] = i
	m.modules[name] = mod
}

// LoadModules initializes and registers the provided integration modules.
func (m *Manager) LoadModules(
	appName string,
	logger *slog.Logger,
	kube *k8s.Kube,
	modules []api.IntegrationModule,
) error {
	for _, mod := range modules {
		impl := mod.Init(logger, kube)

		secretName := fmt.Sprintf("%s-%s-integration", appName, mod.Name)
		wrapper := integration.NewSecret(logger, kube, secretName, impl)

		m.Register(mod, wrapper)
	}
	return nil
}

// NewManager instantiates a new Manager.
func NewManager() *Manager {
	return &Manager{
		integrations: map[IntegrationName]*integration.Integration{},
		modules:      map[IntegrationName]api.IntegrationModule{},
	}
}

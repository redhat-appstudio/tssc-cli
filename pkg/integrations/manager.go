package integrations

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/redhat-appstudio/tssc-cli/pkg/config"
	"github.com/redhat-appstudio/tssc-cli/pkg/constants"
	"github.com/redhat-appstudio/tssc-cli/pkg/integration"
	"github.com/redhat-appstudio/tssc-cli/pkg/k8s"
)

// IntegrationName name of a integration.
type IntegrationName string

// Manager represents the actor responsible for all integrations. It centralizes
// the management of integration instances, keeping a consistent set of
// integration names.
type Manager struct {
	integrations map[IntegrationName]*integration.Integration // integrations
}

const (
	ACS            IntegrationName = "acs"
	Artifactory    IntegrationName = "artifactory"
	Azure          IntegrationName = "azure"
	BitBucket      IntegrationName = "bitbucket"
	GitHub         IntegrationName = "github"
	GitLab         IntegrationName = "gitlab"
	Jenkins        IntegrationName = "jenkins"
	Nexus          IntegrationName = "nexus"
	Quay           IntegrationName = "quay"
	Trustification IntegrationName = "trustification"
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

// NewManager instantiates a new Manager.
func NewManager(logger *slog.Logger, kube *k8s.Kube) *Manager {
	m := &Manager{integrations: map[IntegrationName]*integration.Integration{}}

	// Instantiating all integrations making sure the set of integrations is
	// complete and unique. The application must panic on duplicated integrations.
	for name, data := range map[IntegrationName]integration.Interface{
		ACS:            integration.NewACS(),
		Artifactory:    integration.NewContainerRegistry(""),
		Azure:          integration.NewAzure(),
		BitBucket:      integration.NewBitBucket(),
		GitHub:         integration.NewGitHub(logger, kube),
		GitLab:         integration.NewGitLab(logger),
		Jenkins:        integration.NewJenkins(),
		Nexus:          integration.NewContainerRegistry(""),
		Quay:           integration.NewContainerRegistry(integration.QuayURL),
		Trustification: integration.NewTrustification(),
	} {
		// Ensure unique integration names.
		if _, exists := m.integrations[name]; exists {
			panic(fmt.Sprintf("integration instance already exists: %q", name))
		}
		// Generating the secret name for each integration.
		secretName := fmt.Sprintf("%s-%s-integration", constants.AppName, name)
		// Creating a new instance of the integration.
		m.integrations[name] = integration.NewSecret(
			logger, kube, secretName, data)
	}

	return m
}

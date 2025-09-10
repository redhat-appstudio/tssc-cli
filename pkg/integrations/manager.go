package integrations

import (
	"fmt"
	"log/slog"

	"github.com/redhat-appstudio/tssc-cli/pkg/integration"
	"github.com/redhat-appstudio/tssc-cli/pkg/k8s"
)

// Manager represents the actor responsible for all integrations. It centralizes
// the management of integration instances, keeping a consistent set of
// integration names.
type Manager struct {
	integrations map[string]*integration.Integration // integration handlers
}

// IntegrationName name of a integration.
type IntegrationName string

const (
	ACS            IntegrationName = "acs"
	Artifactory    IntegrationName = "artifactory"
	Azure          IntegrationName = "azure"
	BitBucket      IntegrationName = "bitbucket"
	GitHubApp      IntegrationName = "github-app"
	GitLab         IntegrationName = "gitlab"
	Jenkins        IntegrationName = "jenkins"
	Nexus          IntegrationName = "nexus"
	Quay           IntegrationName = "quay"
	Trustification IntegrationName = "trustification"
)

const (
	// publicQuayURL is the default URL for public Quay.
	publicQuayURL = "https://quay.io"
)

func (m *Manager) Get(name IntegrationName) *integration.Integration {
	is, exists := m.integrations[string(name)]
	if !exists {
		panic(fmt.Sprintf("integration instance is not found: %q", name))
	}
	return is
}

// NewManager instantiates a new Manager.
func NewManager(logger *slog.Logger, kube *k8s.Kube) *Manager {
	m := &Manager{integrations: map[string]*integration.Integration{}}

	// Instantiating all integrations making sure the set of integrations is
	// complete and unique. The application must panic on duplicated integrations.
	for name, data := range map[IntegrationName]integration.Interface{
		ACS:            integration.NewACS(),
		Artifactory:    integration.NewContainerRegistry(""),
		Azure:          integration.NewAzure(),
		BitBucket:      integration.NewBitBucket(),
		GitHubApp:      integration.NewGitHubApp(logger, kube),
		GitLab:         integration.NewGitLab(logger),
		Jenkins:        integration.NewJenkins(),
		Nexus:          integration.NewContainerRegistry(""),
		Quay:           integration.NewContainerRegistry(publicQuayURL),
		Trustification: integration.NewTrustification(),
	} {
		if _, exists := m.integrations[string(name)]; exists {
			panic(fmt.Sprintf("integration instance already exists: %q", name))
		}
		m.integrations[string(name)] = integration.NewSecret(logger, kube, data)
	}

	return m
}

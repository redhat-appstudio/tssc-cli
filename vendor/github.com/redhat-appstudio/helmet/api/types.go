package api

import (
	"log/slog"

	"github.com/redhat-appstudio/helmet/internal/integration"
	"github.com/redhat-appstudio/helmet/internal/k8s"
)

// IntegrationModule defines the contract for a pluggable integration.
// It encapsulates both the integration business logic (integration.Interface) and
// the CLI representation (SubCommand).
type IntegrationModule struct {
	// Name is the unique name of the integration (e.g., "github", "acs").
	Name string

	// Init creates the integration business logic instance.
	Init func(*slog.Logger, *k8s.Kube) integration.Interface

	// Command creates the CLI subcommand for this integration.
	// It receives the application context and initialized integration wrapper.
	Command func(
		*AppContext,
		*slog.Logger,
		*k8s.Kube,
		*integration.Integration,
	) SubCommand
}

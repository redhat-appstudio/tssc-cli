package integration

import (
	"context"
	"log/slog"

	"github.com/redhat-appstudio/helmet/internal/config"

	"github.com/spf13/cobra"
	corev1 "k8s.io/api/core/v1"
)

type Interface interface {
	// PersistentFlags decorates the cobra command with integration flags.
	PersistentFlags(*cobra.Command)

	// LoggerWith decorates the logger with the integration attributes.
	LoggerWith(*slog.Logger) *slog.Logger

	// Validate checks if all required fields are set and valid.
	Validate() error

	// Type shares the Kubernetes secret type for the integration.
	Type() corev1.SecretType

	// SetArgument sets a optional key-value argument to be used in the
	// generation of the Kubernetes secret data.
	SetArgument(string, string) error

	// Data generates the data for the Kubernetes secret, provides the payload
	// that will become the integration secret stored in the cluster.
	Data(context.Context, *config.Config) (map[string][]byte, error)
}

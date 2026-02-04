package integration

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/redhat-appstudio/helmet/internal/config"

	"github.com/spf13/cobra"
	corev1 "k8s.io/api/core/v1"
)

// ACS represents the ACS, Advanced Cluster Security, integration. The coordinates
// to connect TSSC with a external instance of ACS.
type ACS struct {
	endpoint string // ACS service endpoint
	token    string // API token credentials
}

var _ Interface = &ACS{}

// PersistentFlags adds the persistent flags to the informed Cobra command.
func (a *ACS) PersistentFlags(c *cobra.Command) {
	p := c.PersistentFlags()

	p.StringVar(&a.endpoint, "endpoint", a.endpoint,
		"ACS service endpoint, formatted as 'hostname:port'")
	p.StringVar(&a.token, "token", a.token,
		"ACS API token")

	for _, f := range []string{"endpoint", "token"} {
		if err := c.MarkPersistentFlagRequired(f); err != nil {
			panic(err)
		}
	}
}

// SetArgument sets additional arguments to the integration.
func (a *ACS) SetArgument(_, _ string) error {
	return nil
}

// LoggerWith decorates the logger with the integration flags.
func (a *ACS) LoggerWith(logger *slog.Logger) *slog.Logger {
	return logger.With("endpoint", a.endpoint, "token-len", len(a.token))
}

// Type shares the Kubernetes secret type for this integration.
func (a *ACS) Type() corev1.SecretType {
	return corev1.SecretTypeOpaque
}

// Validate check if the ACS endpoint doesn't contain the protocol, and carries
// the port number.
func (a *ACS) Validate() error {
	if strings.Contains(a.endpoint, "://") {
		return fmt.Errorf("the protocol must not be specified: %q", a.endpoint)
	}
	if !strings.Contains(a.endpoint, ":") {
		return fmt.Errorf("the port number must be specified: %q", a.endpoint)
	}
	return nil
}

// Data returns the ACS integration data.
func (a *ACS) Data(
	_ context.Context,
	_ *config.Config,
) (map[string][]byte, error) {
	return map[string][]byte{
		"endpoint": []byte(a.endpoint),
		"token":    []byte(a.token),
	}, nil
}

// NewACS creates a new instance of the ACS integration.l
func NewACS() *ACS {
	return &ACS{}
}

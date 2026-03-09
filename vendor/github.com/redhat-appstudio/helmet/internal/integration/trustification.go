package integration

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/redhat-appstudio/helmet/internal/config"
	"github.com/redhat-appstudio/helmet/internal/runcontext"

	"github.com/spf13/cobra"
	corev1 "k8s.io/api/core/v1"
)

// Trustification represents the coordinates to connect the cluster with remote
// Trustification services.
type Trustification struct {
	bombasticURL              string // URL of the BOMbastic api host
	supportedCycloneDXVersion string // CycloneDX supported version.
}

var _ Interface = &Trustification{}

// PersistentFlags adds the persistent flags to the informed Cobra command.
func (t *Trustification) PersistentFlags(c *cobra.Command) {
	p := c.PersistentFlags()

	p.StringVar(&t.bombasticURL, "bombastic-api-url", t.bombasticURL,
		"URL of the BOMbastic api host "+
			"e.g. https://sbom.trustification.dev)")
	p.StringVar(
		&t.supportedCycloneDXVersion,
		"supported-cyclonedx-version",
		t.supportedCycloneDXVersion,
		"If the SBOM uses a higher CycloneDX version, Syft convert to the "+
			"supported version before uploading.",
	)

	if err := c.MarkPersistentFlagRequired("bombastic-api-url"); err != nil {
		panic(err)
	}
}

// SetArgument sets additional arguments to the integration.
func (t *Trustification) SetArgument(string, string) error {
	return nil
}

// LoggerWith decorates the logger with the integration flags.
func (t *Trustification) LoggerWith(logger *slog.Logger) *slog.Logger {
	return logger.With(
		"bombastic-api-url", t.bombasticURL,
		"supported-cyclonedx-version", t.supportedCycloneDXVersion,
	)
}

// Type shares the Kubernetes secret type for this integration.
func (t *Trustification) Type() corev1.SecretType {
	return corev1.SecretTypeOpaque
}

// Validate checks the informed URLs ensure valid inputs.
func (t *Trustification) Validate() error {
	if t.bombasticURL == "" {
		return fmt.Errorf("bombastic-api-url is required")
	}
	var err error
	if err = ValidateURL(t.bombasticURL); err != nil {
		return fmt.Errorf("%w: %q", err, t.bombasticURL)
	}
	return nil
}

// Data returns the Kubernetes secret data for this integration.
func (t *Trustification) Data(
	_ context.Context,
	_ *runcontext.RunContext,
	_ *config.Config,
) (map[string][]byte, error) {
	return map[string][]byte{
		"bombastic_api_url":           []byte(t.bombasticURL),
		"supported_cyclonedx_version": []byte(t.supportedCycloneDXVersion),
	}, nil
}

// NewTrustification creates a new instance of the Trustification integration.
func NewTrustification() *Trustification {
	return &Trustification{}
}

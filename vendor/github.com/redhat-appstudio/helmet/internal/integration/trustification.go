package integration

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/redhat-appstudio/helmet/internal/config"

	"github.com/spf13/cobra"
	corev1 "k8s.io/api/core/v1"
)

// Trustification represents the coordinates to connect the cluster with remote
// Trustification services.
type Trustification struct {
	bombasticURL              string // URL of the BOMbastic api host
	oidcIssuerURL             string // URL of the OIDC token issuer
	oidcClientID              string // OIDC client ID
	oidcClientSecret          string // OIDC client secret
	supportedCycloneDXVersion string // CycloneDX supported version.
}

var _ Interface = &Trustification{}

// PersistentFlags adds the persistent flags to the informed Cobra command.
func (t *Trustification) PersistentFlags(c *cobra.Command) {
	p := c.PersistentFlags()

	p.StringVar(&t.bombasticURL, "bombastic-api-url", t.bombasticURL,
		"URL of the BOMbastic api host "+
			"e.g. https://sbom.trustification.dev)")
	p.StringVar(&t.oidcIssuerURL, "oidc-issuer-url", t.oidcIssuerURL,
		"URL of the OIDC token issuer "+
			"(e.g. https://sso.trustification.dev/realms/chicken)")
	p.StringVar(&t.oidcClientID, "oidc-client-id", t.oidcClientID,
		"OIDC client ID")
	p.StringVar(&t.oidcClientSecret, "oidc-client-secret", t.oidcClientSecret,
		"OIDC client secret")
	p.StringVar(
		&t.supportedCycloneDXVersion,
		"supported-cyclonedx-version",
		t.supportedCycloneDXVersion,
		"If the SBOM uses a higher CycloneDX version, Syft convert to the "+
			"supported version before uploading.",
	)

	for _, f := range []string{
		"bombastic-api-url",
		"oidc-issuer-url",
		"oidc-client-id",
		"oidc-client-secret",
	} {
		if err := c.MarkPersistentFlagRequired(f); err != nil {
			panic(err)
		}
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
		"oidc-issuer-url", t.oidcIssuerURL,
		"oidc-client-id", t.oidcClientID,
		"oidc-client-secret-len", len(t.oidcClientSecret),
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
		return fmt.Errorf("%s: %q", err, t.bombasticURL)
	}
	if t.oidcIssuerURL == "" {
		return fmt.Errorf("oidc-issuer-url is required")
	}
	if err = ValidateURL(t.oidcIssuerURL); err != nil {
		return fmt.Errorf("%s: %q", err, t.oidcIssuerURL)
	}
	if t.oidcClientID == "" {
		return fmt.Errorf("oidc-client-id is required")
	}
	if t.oidcClientSecret == "" {
		return fmt.Errorf("oidc-client-secret is required")
	}
	return nil
}

// Data returns the Kubernetes secret data for this integration.
func (t *Trustification) Data(
	_ context.Context,
	_ *config.Config,
) (map[string][]byte, error) {
	return map[string][]byte{
		"bombastic_api_url":           []byte(t.bombasticURL),
		"oidc_client_id":              []byte(t.oidcClientID),
		"oidc_client_secret":          []byte(t.oidcClientSecret),
		"oidc_issuer_url":             []byte(t.oidcIssuerURL),
		"supported_cyclonedx_version": []byte(t.supportedCycloneDXVersion),
	}, nil
}

// NewTrustification creates a new instance of the Trustification integration.
func NewTrustification() *Trustification {
	return &Trustification{}
}

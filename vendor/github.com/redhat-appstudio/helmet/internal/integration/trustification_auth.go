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

// TrustificationAuth represents the authentication to connect the cluster with remote
// Trustification services.
type TrustificationAuth struct {
	oidcIssuerURL    string // URL of the OIDC token issuer
	oidcClientID     string // OIDC client ID
	oidcClientSecret string // OIDC client secret
}

var _ Interface = &TrustificationAuth{}

// PersistentFlags adds the persistent flags to the informed Cobra command.
func (t *TrustificationAuth) PersistentFlags(c *cobra.Command) {
	p := c.PersistentFlags()

	p.StringVar(&t.oidcIssuerURL, "oidc-issuer-url", t.oidcIssuerURL,
		"URL of the OIDC token issuer "+
			"(e.g. https://sso.trustification.dev/realms/chicken)")
	p.StringVar(&t.oidcClientID, "oidc-client-id", t.oidcClientID,
		"OIDC client ID")
	p.StringVar(&t.oidcClientSecret, "oidc-client-secret", t.oidcClientSecret,
		"OIDC client secret")

	for _, f := range []string{
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
func (t *TrustificationAuth) SetArgument(string, string) error {
	return nil
}

// LoggerWith decorates the logger with the integration flags.
func (t *TrustificationAuth) LoggerWith(logger *slog.Logger) *slog.Logger {
	return logger.With(
		"oidc-issuer-url", t.oidcIssuerURL,
		"oidc-client-id", t.oidcClientID,
		"oidc-client-secret-len", len(t.oidcClientSecret),
	)
}

// Type shares the Kubernetes secret type for this integration.
func (t *TrustificationAuth) Type() corev1.SecretType {
	return corev1.SecretTypeOpaque
}

// Validate checks the informed URLs ensure valid inputs.
func (t *TrustificationAuth) Validate() error {
	var err error
	if t.oidcIssuerURL == "" {
		return fmt.Errorf("oidc-issuer-url is required")
	}
	if err = ValidateURL(t.oidcIssuerURL); err != nil {
		return fmt.Errorf("%w: %q", err, t.oidcIssuerURL)
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
func (t *TrustificationAuth) Data(
	_ context.Context,
	_ *runcontext.RunContext,
	_ *config.Config,
) (map[string][]byte, error) {
	return map[string][]byte{
		"oidc_client_id":     []byte(t.oidcClientID),
		"oidc_client_secret": []byte(t.oidcClientSecret),
		"oidc_issuer_url":    []byte(t.oidcIssuerURL),
	}, nil
}

// NewTrustificationAuth creates a new instance of the TrustificationAuth integration.
func NewTrustificationAuth() *TrustificationAuth {
	return &TrustificationAuth{}
}

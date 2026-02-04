package integration

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/redhat-appstudio/helmet/internal/config"

	"github.com/spf13/cobra"
	corev1 "k8s.io/api/core/v1"
)

// TrustedArtifactSigner represents the coordinates to connect to
// the TrustedArtifactSigner services.
type TrustedArtifactSigner struct {
	rekorURL string // URL of the rekor server
	tufURL   string // URL of the TUF server
}

var _ Interface = &TrustedArtifactSigner{}

// PersistentFlags adds the persistent flags to the informed Cobra command.
func (t *TrustedArtifactSigner) PersistentFlags(c *cobra.Command) {
	p := c.PersistentFlags()

	p.StringVar(&t.rekorURL, "rekor-url", t.rekorURL,
		"URL of the rekor server "+
			"(e.g. https://rekor.sigstore.dev)")
	p.StringVar(&t.tufURL, "tuf-url", t.tufURL,
		"URL of the TUF server "+
			"(e.g. https://tuf.trustification.dev)")

	for _, f := range []string{
		"rekor-url",
		"tuf-url",
	} {
		if err := c.MarkPersistentFlagRequired(f); err != nil {
			panic(err)
		}
	}
}

// SetArgument sets additional arguments to the integration.
func (t *TrustedArtifactSigner) SetArgument(string, string) error {
	return nil
}

// LoggerWith decorates the logger with the integration flags.
func (t *TrustedArtifactSigner) LoggerWith(logger *slog.Logger) *slog.Logger {
	return logger.With(
		"rekor-url", t.rekorURL,
		"tuf-url", t.tufURL,
	)
}

// Type shares the Kubernetes secret type for this integration.
func (t *TrustedArtifactSigner) Type() corev1.SecretType {
	return corev1.SecretTypeOpaque
}

// Validate checks the informed URLs ensure valid inputs.
func (t *TrustedArtifactSigner) Validate() error {
	if t.rekorURL == "" {
		return fmt.Errorf("rekor-url is required")
	}
	var err error
	if err = ValidateURL(t.rekorURL); err != nil {
		return fmt.Errorf("%s: %q", err, t.rekorURL)
	}
	if t.tufURL == "" {
		return fmt.Errorf("tuf-url is required")
	}
	if err = ValidateURL(t.tufURL); err != nil {
		return fmt.Errorf("%s: %q", err, t.tufURL)
	}
	return nil
}

// Data returns the Kubernetes secret data for this integration.
func (t *TrustedArtifactSigner) Data(
	_ context.Context,
	_ *config.Config,
) (map[string][]byte, error) {
	return map[string][]byte{
		"rekor_url": []byte(t.rekorURL),
		"tuf_url":   []byte(t.tufURL),
	}, nil
}

// NewTrustedArtifactSigner creates a new instance of the TrustedArtifactSigner integration.
func NewTrustedArtifactSigner() *TrustedArtifactSigner {
	return &TrustedArtifactSigner{}
}

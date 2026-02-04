package integration

import (
	"context"
	"log/slog"

	"github.com/redhat-appstudio/helmet/internal/config"

	"github.com/spf13/cobra"
	corev1 "k8s.io/api/core/v1"
)

// Jenkins represents the Jenkins integration coordinates.
type Jenkins struct {
	url      string // Jenkins service URL
	username string // user to connect to the service
	token    string // API token credentials
}

var _ Interface = &Jenkins{}

// PersistentFlags adds the persistent flags to the informed Cobra command.
func (j *Jenkins) PersistentFlags(c *cobra.Command) {
	p := c.PersistentFlags()

	p.StringVar(&j.url, "url", j.url,
		"Jenkins URL to the service")
	p.StringVar(&j.username, "username", j.username,
		"Jenkins user to connect to the service")
	p.StringVar(&j.token, "token", j.token,
		"Jenkins API token")

	for _, f := range []string{"token", "username", "url"} {
		if err := c.MarkPersistentFlagRequired(f); err != nil {
			panic(err)
		}
	}
}

// SetArgument sets additional arguments to the integration.
func (j *Jenkins) SetArgument(string, string) error {
	return nil
}

// LoggerWith decorates the logger with the integration flags.
func (j *Jenkins) LoggerWith(logger *slog.Logger) *slog.Logger {
	return logger.With(
		"url", j.url,
		"username", j.username,
		"token-len", len(j.token),
	)
}

// Type returns the type of the integration.
func (j *Jenkins) Type() corev1.SecretType {
	return corev1.SecretTypeOpaque
}

// Validate validates the integration configuration.
func (j *Jenkins) Validate() error {
	return ValidateURL(j.url)
}

// Data returns the integration data for Jenkins.
func (j *Jenkins) Data(
	_ context.Context,
	_ *config.Config,
) (map[string][]byte, error) {
	return map[string][]byte{
		"baseUrl":  []byte(j.url),
		"token":    []byte(j.token),
		"username": []byte(j.username),
	}, nil
}

// NewJenkins instantiates a new Jenkins integration.
func NewJenkins() *Jenkins {
	return &Jenkins{}
}

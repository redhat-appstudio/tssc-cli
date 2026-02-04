package integration

import (
	"context"
	"log/slog"

	"github.com/redhat-appstudio/helmet/internal/config"

	"github.com/spf13/cobra"
	corev1 "k8s.io/api/core/v1"
)

// BitBucket represents the BitBucket integration coordinates.
type BitBucket struct {
	appPassword string // password
	host        string // endpoint
	username    string // username
}

var _ Interface = &BitBucket{}

// PersistentFlags adds the persistent flags to the informed Cobra command.
func (b *BitBucket) PersistentFlags(c *cobra.Command) {
	p := c.PersistentFlags()

	p.StringVar(&b.host, "host", b.host,
		"BitBucket API endpoint")
	p.StringVar(&b.username, "username", b.username,
		"BitBucket username")
	p.StringVar(&b.appPassword, "app-password", b.appPassword,
		"BitBucket application password")

	for _, f := range []string{"host", "username", "app-password"} {
		if err := c.MarkPersistentFlagRequired(f); err != nil {
			panic(err)
		}
	}
}

// SetArgument sets additional arguments to the integration.
func (b *BitBucket) SetArgument(_, _ string) error {
	return nil
}

// LoggerWith decorates the logger with the integration flags.
func (b *BitBucket) LoggerWith(logger *slog.Logger) *slog.Logger {
	return logger.With(
		"username", b.username,
		"app-password-len", len(b.appPassword),
		"host", b.host,
	)
}

// Validate validates the integration.
func (b *BitBucket) Validate() error {
	return nil
}

// Type shares the Kubernetes secret type for this integration.
func (b *BitBucket) Type() corev1.SecretType {
	return corev1.SecretTypeOpaque
}

// Data returns the BitBucket integration data.
func (b *BitBucket) Data(
	_ context.Context,
	_ *config.Config,
) (map[string][]byte, error) {
	return map[string][]byte{
		"host":        []byte(b.host),
		"username":    []byte(b.username),
		"appPassword": []byte(b.appPassword),
	}, nil
}

// NewBitBucket creates a new BitBucket integration instance. By default it uses
// the public BitBucket host.
func NewBitBucket() *BitBucket {
	return &BitBucket{host: "bitbucket.org"}
}

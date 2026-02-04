package integration

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/redhat-appstudio/helmet/internal/config"

	"github.com/spf13/cobra"
	corev1 "k8s.io/api/core/v1"
)

// Azure represents the Azure integration coordinates.
type Azure struct {
	host         string // azure host
	token        string // api token credentials
	org          string // azure organization name
	clientID     string // azure client id
	clientSecret string // azure client secret
	tenantID     string // azure tenant id
}

var _ Interface = &Azure{}

// PersistentFlags adds the persistent flags to the informed Cobra command.
func (a *Azure) PersistentFlags(c *cobra.Command) {
	p := c.PersistentFlags()

	p.StringVar(&a.host, "host", a.host,
		"Azure host")
	p.StringVar(&a.token, "token", a.token,
		"Azure API token")
	p.StringVar(&a.org, "organization", a.org,
		"Azure organization name")
	p.StringVar(&a.clientID, "client-id", a.clientID,
		"Azure client ID")
	p.StringVar(&a.clientSecret, "client-secret", a.clientSecret,
		"Azure client secret")
	p.StringVar(&a.tenantID, "tenant-id", a.tenantID,
		"Azure tenant ID")

	for _, f := range []string{"host", "organization", "token"} {
		if err := c.MarkPersistentFlagRequired(f); err != nil {
			panic(err)
		}
	}
}

// SetArgument sets additional arguments to the integration.
func (a *Azure) SetArgument(string, string) error {
	return nil
}

// LoggerWith decorates the logger with the integration flags.
func (a *Azure) LoggerWith(logger *slog.Logger) *slog.Logger {
	return logger.With(
		"host", a.host,
		"token-len", len(a.token),
		"organization", a.org,
		"client-id", a.clientID,
		"client-secret-len", len(a.clientSecret),
		"tenant-id-len", len(a.tenantID),
	)
}

// Validate ensures the Azure flags are valid.
func (a *Azure) Validate() error {
	if a.clientID == "" && (a.clientSecret != "" || a.tenantID != "") {
		return fmt.Errorf(
			"client-id is required when client-secret or tenant-id is specified")
	}
	if a.clientSecret == "" && a.tenantID != "" {
		return fmt.Errorf("client-secret is required when tenant-id is specified")
	}
	if a.clientSecret != "" && a.tenantID == "" {
		return fmt.Errorf("tenant-id is required when client-secret is specified")
	}
	return nil
}

// Type returns the type of the integration.
func (a *Azure) Type() corev1.SecretType {
	return corev1.SecretTypeOpaque
}

// Data returns the data for the Azure integration secret.
func (a *Azure) Data(context.Context, *config.Config) (map[string][]byte, error) {
	return map[string][]byte{
		"host":         []byte(a.host),
		"token":        []byte(a.token),
		"organization": []byte(a.org),
		"clientId":     []byte(a.clientID),
		"clientSecret": []byte(a.clientSecret),
		"tenantId":     []byte(a.tenantID),
	}, nil
}

// NewAzure creates a new Azure integration instance with default public host.
func NewAzure() *Azure {
	return &Azure{
		host: "dev.azure.com",
	}
}

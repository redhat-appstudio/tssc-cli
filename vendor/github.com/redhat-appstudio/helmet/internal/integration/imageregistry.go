package integration

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/redhat-appstudio/helmet/internal/config"

	"github.com/spf13/cobra"
	corev1 "k8s.io/api/core/v1"
)

// ImageRegistry represents the image registry integration coordinates. Works with
// different TSSC integrations managing image registry configuration.
type ImageRegistry struct {
	dockerConfig   string // registry credentials (JSON)
	dockerConfigRO string // registry read-only credentials (JSON)
	url            string // API endpoint
	token          string // API token
	organization   string // optional: Quay organization name for additional token secret
}

var _ Interface = &ImageRegistry{}

const (
	// QuayURL is the default URL for public Quay.
	QuayURL = "https://quay.io"

	// dockerConfigEx is an example of a docker config JSON.
	dockerConfigEx = `{ "auths": { "registry.tld": { "auth": "username" } } }`
)

// PersistentFlags adds the persistent flags to the informed Cobra command.
func (i *ImageRegistry) PersistentFlags(cmd *cobra.Command) {
	p := cmd.PersistentFlags()

	p.StringVar(&i.dockerConfig, "dockerconfigjson", i.dockerConfig,
		fmt.Sprintf("JSON formatted registry credentials, e.g.: %q",
			dockerConfigEx))
	p.StringVar(
		&i.dockerConfigRO, "dockerconfigjsonreadonly", i.dockerConfigRO,
		fmt.Sprintf("JSON formatted read-only registry credentials, e.g.: %q",
			dockerConfigEx))
	p.StringVar(&i.url, "url", i.url, "Container registry API endpoint.")
	p.StringVar(&i.token, "token", i.token, "Container registry API token.")
	p.StringVar(&i.organization, "organization", i.organization,
		"Quay organization name.")

	for _, f := range []string{"url"} {
		if err := cmd.MarkPersistentFlagRequired(f); err != nil {
			panic(err)
		}
	}
}

// SetArgument sets additional arguments to the integration.
func (i *ImageRegistry) SetArgument(_, _ string) error {
	return nil
}

// LoggerWith decorates the logger with the integration flags.
func (i *ImageRegistry) LoggerWith(logger *slog.Logger) *slog.Logger {
	return logger.With(
		"dockerconfigjson-len", len(i.dockerConfig),
		"dockerconfigjsonreadonly-len", len(i.dockerConfigRO),
		"url", i.url,
		"token-len", len(i.token),
		"organization", i.organization,
	)
}

// Validate validates the integration configuration.
func (i *ImageRegistry) Validate() error {
	if i.dockerConfig != "" {
		if err := ValidateJSON("dockerconfigjson", i.dockerConfig); err != nil {
			return err
		}
	}
	if i.dockerConfigRO != "" {
		if err := ValidateJSON("dockerconfigjsonreadonly", i.dockerConfigRO); err != nil {
			return err
		}
	}
	return ValidateURL(i.url)
}

// Type returns the type of the integration.
func (i *ImageRegistry) Type() corev1.SecretType {
	if i.dockerConfig != "" {
		return corev1.SecretTypeDockerConfigJson
	}
	return corev1.SecretTypeOpaque
}

// Data returns the integration data.
func (i *ImageRegistry) Data(
	_ context.Context,
	_ *config.Config,
) (map[string][]byte, error) {
	return map[string][]byte{
		".dockerconfigjson":         []byte(i.dockerConfig),
		".dockerconfigjsonreadonly": []byte(i.dockerConfigRO),
		"url":                       []byte(i.url),
		"token":                     []byte(i.token),
		"organization":              []byte(i.organization),
	}, nil
}

// NewContainerRegistry creates a new instance with the default URL.
func NewContainerRegistry(defaultURL string) *ImageRegistry {
	return &ImageRegistry{url: defaultURL}
}

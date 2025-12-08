package config

import (
	"fmt"
	"regexp"
	"strings"
)

const (
	// DeveloperHub Red Hat Developer Hub (RHDH).
	DeveloperHub = "Developer Hub"
	// OpenShiftPipelines OpenShift Pipelines.
	OpenShiftPipelines = "OpenShift Pipelines"
)

// ProductSpec contains the configuration for a specific product.
type Product struct {
	// Name of the product.
	Name string `yaml:"name"`
	// Enabled product toggle.
	Enabled bool `yaml:"enabled"`
	// Namespace target namespace for product's dependency (Helm chart). If empty,
	// it defaults to the installer's namespace.
	Namespace *string `yaml:"namespace,omitempty"`
	// Properties contains the product specific configuration.
	Properties map[string]interface{} `yaml:"properties"`
}

// KeyName returns a sanitized key name for the product.
func (p *Product) KeyName() string {
	// Replace any character that is not a letter, digit, or underscore with a
	// single underscore.
	reg := regexp.MustCompile(`[^a-zA-Z0-9_]+`)
	key := reg.ReplaceAllString(p.Name, "_")

	// Remove leading/trailing underscores that might result from the replacement.
	key = strings.Trim(key, "_")

	// Collapse multiple underscores into a single one.
	key = regexp.MustCompile(`_+`).ReplaceAllString(key, "_")

	// Ensure the name doesn't start with a digit by prefixing it with an
	// underscore if it does.
	if len(key) > 0 && '0' <= key[0] && key[0] <= '9' {
		key = "_" + key
	}
	return key
}

// GetNamespace returns the product namespace, or an empty string if not set.
func (p *Product) GetNamespace() string {
	if p.Namespace == nil {
		return ""
	}
	return *p.Namespace
}

// Validate validates the product configuration, checking for missing fields.
func (p *Product) Validate() error {
	if p.Enabled && p.GetNamespace() == "" {
		return fmt.Errorf("%w: product %q: missing namespace",
			ErrInvalidConfig, p.Name)
	}
	return nil
}

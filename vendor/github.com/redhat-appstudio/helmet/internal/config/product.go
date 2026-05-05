package config

import (
	"fmt"
	"regexp"
	"strings"
)

// productYAML is the wire shape for merged/saved YAML: omit "enabled" unless the
// user has explicitly opted out (false). Absent or true means active without a key.
type productYAML struct {
	Name       string                 `yaml:"name"`
	Enabled    *bool                  `yaml:"enabled,omitempty"`
	Namespace  *string                `yaml:"namespace,omitempty"`
	Properties map[string]interface{} `yaml:"properties,omitempty"`
}

// ProductSpec contains the configuration for a specific product.
type Product struct {
	// Name of the product.
	Name string `yaml:"name"`
	// Enabled, when set to false, excludes the product from deployment while
	// keeping its entry in the config (e.g. MCP / integration flows). When nil
	// or omitted in YAML, the product is active—listing in the installer config
	// is the primary enablement signal.
	Enabled *bool `yaml:"enabled,omitempty"`
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

// MarshalYAML emits optional "enabled" only when explicitly false. Omitted key and
// true both mean active; this keeps default merged config free of enabled flags.
func (p Product) MarshalYAML() (interface{}, error) {
	out := productYAML{
		Name:       p.Name,
		Namespace:  p.Namespace,
		Properties: p.Properties,
	}
	if p.Enabled != nil && !*p.Enabled {
		v := false
		out.Enabled = &v
	}
	return out, nil
}

// IsActive reports whether this product should be deployed. Omitted or nil
// Enabled means true.
func (p *Product) IsActive() bool {
	return p.Enabled == nil || *p.Enabled
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
	if !p.IsActive() {
		return nil
	}
	if p.GetNamespace() == "" {
		return fmt.Errorf("%w: product %q: missing namespace",
			ErrInvalidConfig, p.Name)
	}
	return nil
}

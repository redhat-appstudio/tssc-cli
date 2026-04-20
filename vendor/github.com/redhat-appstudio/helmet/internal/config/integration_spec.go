package config

import (
	"fmt"
	"strings"
)

// IntegrationSpec is one entry under the installer `integrations` list in
// config.yaml. ID is the stable key for templates (matches charts/tssc-<id> and
// local://<id> in helmet.yaml). The deprecated Integration field is accepted as an
// alias when ID is empty.
type IntegrationSpec struct {
	ID          string                 `yaml:"id,omitempty"`
	Integration string                 `yaml:"integration,omitempty"` // deprecated: use id
	Name        string                 `yaml:"name"`
	Properties  map[string]interface{} `yaml:"properties,omitempty"`
}

// EffectiveID returns the machine id for this entry, preferring id over the
// legacy integration field.
func (i IntegrationSpec) EffectiveID() string {
	if s := strings.TrimSpace(i.ID); s != "" {
		return s
	}
	return strings.TrimSpace(i.Integration)
}

// Validate checks a single integration entry.
func (i IntegrationSpec) Validate() error {
	if strings.TrimSpace(i.Name) == "" {
		return fmt.Errorf("%w: integration entry missing name", ErrInvalidConfig)
	}
	if strings.TrimSpace(i.EffectiveID()) == "" {
		return fmt.Errorf("%w: integration entry missing id", ErrInvalidConfig)
	}
	return nil
}

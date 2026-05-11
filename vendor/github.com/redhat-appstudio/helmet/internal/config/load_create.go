package config

import (
	"github.com/redhat-appstudio/helmet/internal/chartfs"
)

// CreateConfigLoader loads installer configuration used by the "config --create"
// command. Implementations may merge distributed fragments when explicitPath is
// empty; otherwise they should load that path from ChartFS.
type CreateConfigLoader func(
	chartFS *chartfs.ChartFS,
	explicitPath string,
	namespace string,
	appIdentifier string,
) (*Config, error)

// DefaultCreateConfigLoader loads a single YAML file (embedded or overlay).
// When explicitPath is empty it uses DefaultRelativeConfigPath.
func DefaultCreateConfigLoader(
	cfs *chartfs.ChartFS,
	explicitPath string,
	namespace string,
	appIdentifier string,
) (*Config, error) {
	path := explicitPath
	if path == "" {
		path = DefaultRelativeConfigPath
	}
	return NewConfigFromFile(cfs, path, namespace, appIdentifier)
}

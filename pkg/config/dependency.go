package config

import (
	"log/slog"
)

// Dependency contains a individual Helm chart configuration.
type Dependency struct {
	// Chart relative location to the Helm chart directory.
	Chart string `yaml:"chart"`
	// Namespace where the Helm chart will be deployed.
	Namespace string `yaml:"namespace"`
	// Enabled Helm Chart toggle.
	Enabled bool `yaml:"enabled"`
}

// LoggerWith decorates the logger with dependency information.
func (d *Dependency) LoggerWith(logger *slog.Logger) *slog.Logger {
	return logger.With(
		"dep-chart", d.Chart,
		"dep-namespace", d.Namespace,
		"dep-enabled", d.Enabled,
	)
}

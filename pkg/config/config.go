package config

import (
	"errors"
	"fmt"
	"log/slog"

	"github.com/redhat-appstudio/rhtap-cli/pkg/chartfs"

	"gopkg.in/yaml.v3"
)

// Spec contains all configuration sections.
type Spec struct {
	// Namespace installer's namespace, where the installer's resources will be
	// deployed. Note, Helm charts deployed by the installer are likely to use a
	// different namespace.
	Namespace string `yaml:"namespace"`
	// Features contains the configuration for the installer features.
	Features map[string]FeatureSpec `yaml:"features"`
	// Dependencies contains the installer Helm chart dependencies.
	Dependencies []Dependency `yaml:"dependencies"`
}

// Config root configuration structure.
type Config struct {
	cfs     *chartfs.ChartFS // embedded filesystem
	payload []byte           // original configuration payload

	Installer Spec `yaml:"rhtapCLI"` // root configuration for the installer
}

var (
	// ErrInvalidConfig indicates the configuration content is invalid.
	ErrInvalidConfig = errors.New("invalid configuration")
	// ErrEmptyConfig indicates the configuration file is empty.
	ErrEmptyConfig = errors.New("empty configuration")
	// ErrUnmarshalConfig indicates the configuration file structure is invalid.
	ErrUnmarshalConfig = errors.New("failed to unmarshal configuration")
)

// DefaultRelativeConfigPath default relative path to YAML configuration file.
var DefaultRelativeConfigPath = fmt.Sprintf("installer/%s", Filename)

// GetDependency returns a dependency chart configuration.
func (c *Config) GetDependency(logger *slog.Logger, chart string) (*Dependency, error) {
	logger.Debug("Getting dependency")
	for _, dep := range c.Installer.Dependencies {
		if dep.Chart == chart {
			return &dep, nil
		}
	}
	return nil, fmt.Errorf("chart '%s' not found", chart)
}

// GetEnabledDependencies returns a list of enabled dependencies.
func (c *Config) GetEnabledDependencies(logger *slog.Logger) []Dependency {
	enabled := []Dependency{}
	logger.Debug("Getting enabled dependencies")
	for _, dep := range c.Installer.Dependencies {
		if dep.Enabled {
			logger.Debug("Using dependency...", "dep-chart", dep.Chart)
			enabled = append(enabled, dep)
		} else {
			logger.Debug("Skipping dependency...", "dep-chart", dep.Chart)
		}
	}
	return enabled
}

// GetFeature returns a feature by name, or an error if the feature is not found.
func (c *Config) GetFeature(name string) (*FeatureSpec, error) {
	feature, ok := c.Installer.Features[name]
	if !ok {
		return nil, fmt.Errorf("feature '%s' not found", name)
	}
	return &feature, nil
}

// Validate validates the configuration, checking for missing fields.
func (c *Config) Validate() error {
	root := c.Installer
	// The installer itself must have a namespace.
	if root.Namespace == "" {
		return fmt.Errorf("%w: missing namespace", ErrInvalidConfig)
	}

	// Validating the features, making sure every feature entry is valid.
	for _, feature := range root.Features {
		if err := feature.Validate(); err != nil {
			return err
		}
	}

	// Making sure the installer has a list of dependencies.
	if len(root.Dependencies) == 0 {
		return fmt.Errorf("%w: missing dependencies", ErrInvalidConfig)
	}
	// Validating each dependency, making sure they have the required fields.
	for pos, dep := range root.Dependencies {
		if dep.Chart == "" {
			return fmt.Errorf(
				"%w: missing chart in dependency %d", ErrInvalidConfig, pos)
		}
		if dep.Namespace == "" {
			return fmt.Errorf(
				"%w: missing namespace in dependency %d", ErrInvalidConfig, pos)
		}
	}
	return nil
}

// MarshalYAML marshals the Config into a YAML byte array.
func (c *Config) MarshalYAML() ([]byte, error) {
	return yaml.Marshal(c)
}

// UnmarshalYAML Un-marshals the YAML payload into the Config struct, checking the
// validity of the configuration.
func (c *Config) UnmarshalYAML() error {
	if len(c.payload) == 0 {
		return ErrEmptyConfig
	}
	if err := yaml.Unmarshal(c.payload, c); err != nil {
		return fmt.Errorf("%w: %w", ErrUnmarshalConfig, err)
	}
	return c.Validate()
}

// String returns this configuration as string, indented with two spaces.
func (c *Config) String() string {
	return string(c.payload)
}

// NewConfigFromFile returns a new Config instance based on the informed file.
func NewConfigFromFile(cfs *chartfs.ChartFS, configPath string) (*Config, error) {
	c := &Config{cfs: cfs}
	var err error
	c.payload, err = c.cfs.ReadFile(configPath)
	if err != nil {
		return nil, err
	}
	if err = c.UnmarshalYAML(); err != nil {
		return nil, err
	}
	return c, nil
}

// NewConfigFromBytes instantiates a new Config from the bytes payload informed.
func NewConfigFromBytes(payload []byte) (*Config, error) {
	c := &Config{payload: payload}
	if err := yaml.Unmarshal(payload, c); err != nil {
		return nil, fmt.Errorf("%w: %s", ErrUnmarshalConfig, err)
	}
	return c, nil
}

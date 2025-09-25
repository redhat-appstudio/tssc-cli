package config

import (
	"bytes"
	"errors"
	"fmt"
	"strings"

	"github.com/redhat-appstudio/tssc-cli/pkg/chartfs"

	"gopkg.in/yaml.v3"
)

// Settings represents a map of configuration settings.
type Settings map[string]interface{}

// ProductSpec represents a map of product name and specification.
type Products []Product

// Spec contains all configuration sections.
type Spec struct {
	// Namespace installer's namespace, where the installer's resources will be
	// deployed. Note, Helm charts deployed by the installer are likely to use a
	// different namespace.
	Namespace string `yaml:"namespace"`
	// Settings contains the configuration for the installer settings.
	Settings Settings `yaml:"settings"`
	// Products contains the configuration for the installer products.
	Products Products `yaml:"products"`
}

// Config root configuration structure.
type Config struct {
	cfs       *chartfs.ChartFS // embedded filesystem
	Installer Spec             `yaml:"tssc"` // root configuration for the installer
	root      yaml.Node        // yaml data representation
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

// GetProduct returns a product by name, or an error if the product is not found.
func (c *Config) GetProduct(name string) (*Product, error) {
	for i := range c.Installer.Products {
		if c.Installer.Products[i].Name == name {
			return &c.Installer.Products[i], nil
		}
	}
	return nil, fmt.Errorf("product '%s' not found", name)
}

// GetEnabledProducts returns a map of enabled products.
func (c *Config) GetEnabledProducts() Products {
	enabled := Products{}
	for _, product := range c.Installer.Products {
		if product.Enabled {
			enabled = append(enabled, product)
		}
	}
	return enabled
}

// Validate validates the configuration, checking for missing fields.
func (c *Config) Validate() error {
	root := c.Installer
	// The installer itself must have a namespace.
	if root.Namespace == "" {
		return fmt.Errorf("%w: missing namespace", ErrInvalidConfig)
	}

	// The installer must have a settings section.
	if root.Settings == nil {
		return fmt.Errorf("%w: missing settings", ErrInvalidConfig)
	}

	// Validating the products, making sure every product entry is valid.
	for _, product := range root.Products {
		if err := product.Validate(); err != nil {
			return err
		}
	}
	return nil
}

// DecodeNode returns a struct converted from *yaml.Node
func (c *Config) DecodeNode() error {
	if len(c.root.Content) == 0 {
		return fmt.Errorf("invalid configuration: content is empty")
	}
	doc := c.root.Content[0]
	if doc.Kind != yaml.MappingNode || len(doc.Content) < 2 {
		return fmt.Errorf("invalid configuration: root must be a mapping")
	}
	var tsscNode *yaml.Node
	for i := 0; i+1 < len(doc.Content); i += 2 {
		if doc.Content[i].Value == "tssc" {
			tsscNode = doc.Content[i+1]
			break
		}
	}
	if tsscNode == nil {
		return fmt.Errorf("invalid configuration: missing 'tssc' key")
	}
	if err := tsscNode.Decode(&c.Installer); err != nil {
		return err
	}
	return nil
}

// MarshalYAML marshals the Config into a YAML byte array.
func (c *Config) MarshalYAML() ([]byte, error) {
	var buf bytes.Buffer
	if len(c.root.Content) == 0 {
		return nil, fmt.Errorf("invalid configuration format: content is nil or empty")
	}
	buf.WriteString("---\n")
	encoder := yaml.NewEncoder(&buf)
	encoder.SetIndent(2)
	defer encoder.Close()
	if err := encoder.Encode(c.root.Content[0]); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// UnmarshalYAML Un-marshals the YAML payload into the Config struct, checking the
// validity of the configuration.
func (c *Config) UnmarshalYAML(payload []byte) error {
	if len(payload) == 0 {
		return ErrEmptyConfig
	}
	if err := yaml.Unmarshal(payload, &c.root); err != nil {
		return fmt.Errorf("%w: %w", ErrUnmarshalConfig, err)
	}
	if err := c.DecodeNode(); err != nil {
		return fmt.Errorf("%w: %w", ErrUnmarshalConfig, err)
	}
	return c.Validate()
}

// String returns this configuration as string, indented with two spaces.
func (c *Config) String() string {
	data, err := c.MarshalYAML()
	if err != nil {
		panic(err)
	}
	return string(data)
}

// UpdateMappingValue updates configuration with new value
func (c *Config) UpdateMappingValue(node *yaml.Node, key string, newValue any) error {
	for i := 0; i < len(node.Content); i += 2 {
		if i+1 < len(node.Content) && node.Content[i].Value == key {
			strVal := fmt.Sprintf("%v", newValue)
			node.Content[i+1].Value = strVal
			return nil
		}
	}
	return fmt.Errorf("no key: %s found in configuration", key)
}

// UpdateNestedValue loops in node contents to get the node that needs update
func (c *Config) UpdateNestedValue(node *yaml.Node, path []string, newValue any) error {
	if len(path) == 0 {
		return fmt.Errorf("config path is missing")
	}
	if path[0] == "tssc" {
		path = path[1:]
	}
	if len(path) == 1 {
		return c.UpdateMappingValue(node, path[0], newValue)
	}
	current := node
	key := path[0]
	for i := 0; i < len(current.Content); i += 2 {
		if i+1 < len(current.Content) && current.Content[i].Value == key {
			return c.UpdateNestedValue(current.Content[i+1], path[1:], newValue)
		}
	}
	return fmt.Errorf("not able to update configuration, please check the input")
}

// UpdateNestedValues gets the config content and call UpdateNestedValue to update
func (c *Config) UpdateNestedValues(path []string, newValue any) error {
	if len(c.root.Content) == 0 {
		return fmt.Errorf("invalid configuration format: content is nil or empty")
	}
	doc := c.root.Content[0]
	if len(doc.Content) == 0 {
		return fmt.Errorf("invalid configuration format")
	}
	if doc.Kind != yaml.MappingNode {
		return fmt.Errorf("invalid configuration format: root must be a mapping")
	}
	var tsscNode *yaml.Node
	for i := 0; i+1 < len(doc.Content); i += 2 {
		if doc.Content[i].Value == "tssc" {
			tsscNode = doc.Content[i+1]
			break
		}
	}
	if tsscNode == nil {
		return fmt.Errorf("invalid configuration: missing 'tssc' key")
	}
	switch vt := newValue.(type) {
	case string:
		return c.UpdateNestedValue(tsscNode, path, vt)
	case []any:
		if len(path) != len(vt) {
			return fmt.Errorf("key value do not match")
		}
		for i := 0; i < len(path); i += 1 {
			keyPath := strings.Split(path[i], ".")
			if err := c.UpdateNestedValue(tsscNode, keyPath, vt[i]); err != nil {
				return err
			}
		}
		return nil
	default:
		return fmt.Errorf("not able to update configuration, please check the input")
	}
}

// Set returns new configuration with updates
func (c *Config) Set(key string, configData any) error {
	var err error
	switch cd := configData.(type) {
	case string:
		keyPath := strings.Split(key, ".")
		if len(keyPath) < 2 {
			return fmt.Errorf("invalid key set")
		}
		err = c.UpdateNestedValues(keyPath, cd)
	case map[string]any:
		keyPaths, values := ExtractKeyPathsAndValues(cd, key)
		err = c.UpdateNestedValues(keyPaths, values)
	default:
		return fmt.Errorf("unsupported config data type %T", configData)
	}
	if err != nil {
		return err
	}
	return c.DecodeNode()
}

// NewConfigFromFile returns a new Config instance based on the informed file.
func NewConfigFromFile(cfs *chartfs.ChartFS, configPath string) (*Config, error) {
	c := &Config{cfs: cfs}
	var err error
	payload, err := c.cfs.ReadFile(configPath)
	if err != nil {
		return nil, err
	}
	if err = c.UnmarshalYAML(payload); err != nil {
		return nil, err
	}
	return c, nil
}

// NewConfigFromBytes instantiates a new Config from the bytes payload informed.
func NewConfigFromBytes(payload []byte) (*Config, error) {
	c := &Config{}
	if err := c.UnmarshalYAML(payload); err != nil {
		return nil, err
	}
	return c, nil
}

// NewConfigDefault returns a new Config instance with default values, i.e. the
// configuration payload is loading embedded data.
func NewConfigDefault() (*Config, error) {
	cfs, err := chartfs.NewChartFSForCWD()
	if err != nil {
		return nil, err
	}
	return NewConfigFromFile(cfs, DefaultRelativeConfigPath)
}

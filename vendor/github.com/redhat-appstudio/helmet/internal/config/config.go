package config

import (
	"bytes"
	"errors"
	"fmt"
	"strings"

	"github.com/redhat-appstudio/helmet/internal/chartfs"
	"github.com/redhat-appstudio/helmet/internal/constants"

	"gopkg.in/yaml.v3"
)

// Settings represents a map of configuration settings.
type Settings map[string]interface{}

// ProductSpec represents a map of product name and specification.
type Products []Product

// Spec contains all configuration sections.
type Spec struct {
	// Settings contains the configuration for the installer settings.
	Settings Settings `yaml:"settings"`
	// Products contains the configuration for the installer products.
	Products Products `yaml:"products"`
}

// Config root configuration structure.
type Config struct {
	cfs       *chartfs.ChartFS // embedded filesystem
	root      yaml.Node        // yaml data representation
	namespace string           // installer's namespace

	Installer Spec `yaml:"tssc"` // root configuration for the installer
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
var DefaultRelativeConfigPath = constants.ConfigFilename

// Namespace returns the installer's namespace.
func (c *Config) Namespace() string {
	return c.namespace
}

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

// ApplyDefaults applies default values to the configuration.
func (c *Config) ApplyDefaults() {
	// Propagate the installer namespace to the products.
	for i := range c.Installer.Products {
		if c.Installer.Products[i].Namespace == nil {
			ns := c.namespace
			c.Installer.Products[i].Namespace = &ns
		}
	}
}

// Validate validates the configuration, checking for missing fields.
func (c *Config) Validate() error {
	root := c.Installer

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

// Set returns new configuration with updates
func (c *Config) Set(key string, configData any) error {
	keyPaths, err := FlattenMap(configData, key)
	if err != nil {
		return err
	}

	for keyPath, value := range keyPaths {
		keys := strings.Split(keyPath, ".")
		if err = UpdateNestedValue(&c.root, keys, value); err != nil {
			return err
		}
	}

	return c.DecodeNode()
}

// SetProduct updates an existing product specification in the configuration. It
// searches for a product by its name and, if found, replaces its specification
// with the provided `spec`. The configuration is then re-decoded to reflect the
// changes.
func (c *Config) SetProduct(name string, spec Product) error {
	if len(c.root.Content) == 0 {
		return fmt.Errorf("invalid configuration: content is empty")
	}
	doc := c.root.Content[0]

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

	var productsNode *yaml.Node
	for i := 0; i+1 < len(tsscNode.Content); i += 2 {
		if tsscNode.Content[i].Value == "products" {
			productsNode = tsscNode.Content[i+1]
			break
		}
	}
	if productsNode == nil {
		return fmt.Errorf("invalid configuration: missing 'products' key")
	}

	if productsNode.Kind != yaml.SequenceNode {
		return fmt.Errorf("'products' is not a sequence")
	}

	for i, productNode := range productsNode.Content {
		// Each productNode is a MappingNode
		var productName string
		for j := 0; j+1 < len(productNode.Content); j += 2 {
			if productNode.Content[j].Value == "name" {
				productName = productNode.Content[j+1].Value
				break
			}
		}

		// Found it. Update the node fields in place using Set logic.
		if productName == name {
			// Convert the Product struct spec into a map[stringany for
			// flattening.
			var specMap map[string]any
			data, err := yaml.Marshal(spec)
			if err != nil {
				return fmt.Errorf("failed to marshal product spec: %w", err)
			}
			if err := yaml.Unmarshal(data, &specMap); err != nil {
				return fmt.Errorf("failed to unmarshal product spec: %w", err)
			}

			// Construct the path prefix for this product entry:
			// "tssc.products.[index]".
			pathPrefix := fmt.Sprintf("tssc.products.%d", i)

			keyPaths, err := FlattenMap(specMap, pathPrefix)
			if err != nil {
				return err
			}

			for keyPath, value := range keyPaths {
				keys := strings.Split(keyPath, ".")
				// Skip updating 'name' if it's present in the spec, as it's the
				// lookup key.
				if keys[len(keys)-1] == "name" {
					continue
				}

				if err = UpdateNestedValue(&c.root, keys, value); err != nil {
					return err
				}
			}

			return c.DecodeNode()
		}
	}

	return fmt.Errorf("product %q not found", name)
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
	c.ApplyDefaults()
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

// NewConfigFromFile returns a new Config instance based on the informed file.
func NewConfigFromFile(
	cfs *chartfs.ChartFS,
	configPath string,
	namespace string,
) (*Config, error) {
	c := &Config{cfs: cfs, namespace: namespace}
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
func NewConfigFromBytes(payload []byte, namespace string) (*Config, error) {
	c := &Config{namespace: namespace}
	if err := c.UnmarshalYAML(payload); err != nil {
		return nil, err
	}
	return c, nil
}

// NewConfigDefault returns a new Config instance with default values, i.e. the
// configuration payload is loading embedded data.
func NewConfigDefault(cfs *chartfs.ChartFS, namespace string) (*Config, error) {
	return NewConfigFromFile(cfs, DefaultRelativeConfigPath, namespace)
}

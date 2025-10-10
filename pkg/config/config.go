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
	switch node.Kind {
	case yaml.DocumentNode:
		if len(node.Content) > 0 {
			return c.UpdateMappingValue(node.Content[0], key, newValue)
		}
		// Create root mapping
		mappingNode := &yaml.Node{Kind: yaml.MappingNode, Content: []*yaml.Node{}}
		node.Content = []*yaml.Node{mappingNode}
		return c.UpdateMappingValue(mappingNode, key, newValue)

	case yaml.MappingNode:
		// Find existing key
		for i := 0; i < len(node.Content); i += 2 {
			if node.Content[i].Value == key {
				// Rebuild value node to preserve types.
				var doc yaml.Node
				bs, err := yaml.Marshal(newValue)
				if err != nil {
					return err
				}
				if err := yaml.Unmarshal(bs, &doc); err != nil {
					return err
				}
				if len(doc.Content) == 0 {
					return fmt.Errorf("invalid new value for key %q", key)
				}
				node.Content[i+1] = doc.Content[0]
				return nil
			}
		}
		return nil
	default:
		return fmt.Errorf("cannot set value on node kind: %v", node.Kind)
	}
}

// UpdateNestedValue loops in node contents to get the node that needs update
func (c *Config) UpdateNestedValue(node *yaml.Node, path []string, newValue any) error {
	if len(path) == 0 {
		return fmt.Errorf("config path is missing")
	}
	if len(path) == 1 {
		return c.UpdateMappingValue(node, path[0], newValue)
	}
	key := path[0]
	remainingKeys := path[1:]

	switch node.Kind {
	case yaml.DocumentNode:
		if len(node.Content) > 0 {
			return c.UpdateNestedValue(node.Content[0], path, newValue)
		}
		return fmt.Errorf("invalid config content")

	case yaml.MappingNode:
		for i := 0; i < len(node.Content); i += 2 {
			if strings.EqualFold(node.Content[i].Value, key) {
				return c.UpdateNestedValue(node.Content[i+1], remainingKeys, newValue)
			}
		}
		return fmt.Errorf("key not found: %s", key)

	default:
		return fmt.Errorf("cannot navigate through node kind: %v", node.Kind)
	}
}

// UpdateNestedValues gets the config content and call UpdateNestedValue to update
func (c *Config) UpdateNestedValues(path string, newValue any) error {
	keys := strings.Split(path, ".")
	return c.UpdateNestedValue(&c.root, keys, newValue)
}

func (c *Config) FindNode(node *yaml.Node, key string) (*yaml.Node, error) {
	if key != "products" {
		return nil, fmt.Errorf("key must be 'products'")
	}
	current := node

	switch current.Kind {
	case yaml.DocumentNode:
		if len(current.Content) == 0 {
			return nil, fmt.Errorf("empty document")
		}
		current = current.Content[0]
		return c.FindNode(current, key)
	case yaml.MappingNode:
		for i := 0; i < len(current.Content); i += 2 {
			keyNode := current.Content[i]
			valueNode := current.Content[i+1]
			if keyNode.Value == key {
				return valueNode, nil
			}
		}
		for i := 1; i < len(current.Content); i += 2 {
			result, err := c.FindNode(current.Content[i], key)
			if err == nil && result != nil {
				return result, nil
			}
		}
		return nil, fmt.Errorf("key %q not found", key)
	default:
		return nil, fmt.Errorf("cannot find config: %v", key)
	}
}

// Update product by name (example of custom update logic)
func (c *Config) UpdateProductByName(productName any, keyPath string, value interface{}) error {
	if strings.TrimSpace(keyPath) == "" {
		return fmt.Errorf("config path is missing")
	}
	keys := strings.Split(keyPath, ".")
	pn, ok := productName.(string)
	if !ok {
		return fmt.Errorf("product name must be a string, got %T", productName)
	}
	productsNode, err := c.FindNode(&c.root, "products")
	if err != nil {
		return err
	}
	if productsNode.Kind != yaml.SequenceNode {
		return fmt.Errorf("config of products is not an array")
	}
	for _, productNode := range productsNode.Content {
		if productNode.Kind == yaml.MappingNode {
			for i := 0; i < len(productNode.Content); i += 2 {
				if productNode.Content[i].Value == "name" && strings.EqualFold(productNode.Content[i+1].Value, pn) {
					return c.UpdateNestedValue(productNode, keys, value)
				}
			}
		}
	}
	return fmt.Errorf("product not found: %q", pn)
}

// Set returns new configuration with updates
func (c *Config) Set(key string, configData any) error {
	var keyPaths map[string]any
	var err error
	if strings.Contains(key, "products") {
		prodKeyPaths, err := ExtractProductSettings(configData)
		if err != nil {
			return err
		}
		for _, prodKeyPath := range prodKeyPaths {
			if name, exists := prodKeyPath["name"]; exists {
				delete(prodKeyPath, "name")
				for keyPath, value := range prodKeyPath {
					if err = c.UpdateProductByName(name, keyPath, value); err != nil {
						return err
					}
				}
			} else {
				return fmt.Errorf("product config without name is not supported")
			}
		}
	} else {
		keyPaths, err = FlattenMap(configData, key)
		if err != nil {
			return err
		}
		for keyPath, value := range keyPaths {
			if err = c.UpdateNestedValues(keyPath, value); err != nil {
				return err
			}
		}
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

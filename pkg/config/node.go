package config

import (
	"fmt"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
)

// FindNode searches for a specific key within a YAML node structure.
//
// It traverses the YAML node structure:
//   - If the node is a DocumentNode, it unwraps to its first content node and
//     recurses.
//   - If the node is a MappingNode, it first checks if the key exists directly
//     within the current mapping. If not found, it recursively searches
//     within the values of the current mapping.
//   - For any other node kind, it returns an error.
//
// Returns the yaml.Node corresponding to the found key's value, or an error
// if the key is not found, or the node kind is unsupported.
func FindNode(node *yaml.Node, key string) (*yaml.Node, error) {
	current := node

	switch current.Kind {
	case yaml.DocumentNode:
		if len(current.Content) == 0 {
			return nil, fmt.Errorf("empty document")
		}
		current = current.Content[0]
		return FindNode(current, key)
	case yaml.MappingNode:
		for i := 0; i < len(current.Content); i += 2 {
			keyNode := current.Content[i]
			valueNode := current.Content[i+1]
			if keyNode.Value == key {
				return valueNode, nil
			}
		}
		for i := 1; i < len(current.Content); i += 2 {
			result, err := FindNode(current.Content[i], key)
			if err == nil && result != nil {
				return result, nil
			}
		}
		return nil, fmt.Errorf("key %q not found", key)
	default:
		return nil, fmt.Errorf("cannot find config: %v", key)
	}
}

// UpdateMappingValue updates a key's value within a YAML mapping node.
//
// It traverses the YAML node structure:
//   - If the node is a DocumentNode, it delegates to its first content node.
//     If the DocumentNode is empty, it creates a root mapping node.
//   - If the node is a MappingNode, it searches for the specified key.
//     If found, it marshals the newValue to YAML and unmarshals it back
//     into a new yaml.Node to preserve type fidelity, then replaces the
//     existing value node. If the key is not found, it returns nil.
//   - For any other node kind, it returns an error.
func UpdateMappingValue(node *yaml.Node, key string, newValue any) error {
	switch node.Kind {
	case yaml.DocumentNode:
		if len(node.Content) > 0 {
			return UpdateMappingValue(node.Content[0], key, newValue)
		}
		// Create root mapping.
		mappingNode := &yaml.Node{Kind: yaml.MappingNode, Content: []*yaml.Node{}}
		node.Content = []*yaml.Node{mappingNode}
		return UpdateMappingValue(mappingNode, key, newValue)
	case yaml.MappingNode:
		for i := 0; i < len(node.Content); i += 2 {
			// Find existing key.
			if node.Content[i].Value != key {
				continue
			}

			// Preserve anchor on the old value.
			oldValue := node.Content[i+1]

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

			// Replace node and preserve anchor.
			newValueNode := doc.Content[0]
			newValueNode.Anchor = oldValue.Anchor
			node.Content[i+1] = newValueNode

			return nil
		}
		return nil
	default:
		return fmt.Errorf("cannot set value on node kind: %v", node.Kind)
	}
}

// UpdateNestedValue updates a value deep within a YAML node structure by
// traversing a given path of keys. The path of keys represents the hierarchy of
// the yaml.Node structure.
//
// It handles different node kinds and path lengths:
//   - If the path is empty, it returns an error.
//   - If the path contains only one key, it delegates to UpdateMappingValue.
//   - If the node is a DocumentNode, it unwraps to its first content node and
//     recurses.
//   - If the node is a MappingNode, it iterates through its content to find the
//     first key in the path. If found, it recursively calls itself on the
//     corresponding value node with the rest of the path. If the key is not
//     found, it returns an error.
//   - For any other node kind, it returns an error, as navigation is not
//     possible.
func UpdateNestedValue(node *yaml.Node, keyPath []string, newValue any) error {
	if len(keyPath) == 0 {
		return fmt.Errorf("config path is missing")
	}
	if len(keyPath) == 1 {
		return UpdateMappingValue(node, keyPath[0], newValue)
	}
	key := keyPath[0]
	remainingKeys := keyPath[1:]

	switch node.Kind {
	case yaml.DocumentNode:
		if len(node.Content) > 0 {
			return UpdateNestedValue(node.Content[0], keyPath, newValue)
		}
		return fmt.Errorf("invalid config content")
	case yaml.MappingNode:
		for i := 0; i < len(node.Content); i += 2 {
			if strings.EqualFold(node.Content[i].Value, key) {
				return UpdateNestedValue(
					node.Content[i+1], remainingKeys, newValue)
			}
		}
		return fmt.Errorf("key not found: %s", key)
	case yaml.SequenceNode:
		index, err := strconv.Atoi(key)
		if err != nil {
			return fmt.Errorf("invalid array index: %q", key)
		}
		if index < 0 || index >= len(node.Content) {
			return fmt.Errorf("array index out of bounds: %d", index)
		}
		return UpdateNestedValue(node.Content[index], remainingKeys, newValue)
	default:
		return fmt.Errorf("cannot navigate through node kind: %v", node.Kind)
	}
}

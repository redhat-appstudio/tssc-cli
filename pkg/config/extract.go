package config

import (
	"fmt"
	"maps"
	"strings"
)

var ProdNames = map[string]string{
	"ACS":       "Advanced Cluster Security",
	"TAS":       "Trusted Artifact Signer",
	"GITOPS":    "OpenShift GitOps",
	"DH":        "Developer Hub",
	"TPA":       "Trusted Profile Analyzer",
	"PIPELINES": "OpenShift Pipelines",
}

func ValueExists(m map[string]string, searchValue string) bool {
	for _, value := range m {
		if value == searchValue {
			return true
		}
	}
	return false
}

func ExtractProductSettings(input any) ([]map[string]any, error) {
	var output []map[string]any
	// subConfig := make(map[string]any)
	switch prodConf := input.(type) {
	case map[string]any:
		for key, value := range prodConf {
			item := make(map[string]any)
			keyUpper := strings.ToUpper(key)
			if prodName, exists := ProdNames[keyUpper]; exists {
				item["name"] = prodName
				config, err := FlattenMap(value, "")
				if err != nil {
					return nil, fmt.Errorf("failed to flatten config for product %s: %w", prodName, err)
				}
				if _, ok := config["name"]; ok {
					return nil, fmt.Errorf("invalid product config: top-level key %q is reserved", "name")
				}
				maps.Copy(item, config)
			} else if ValueExists(ProdNames, key) {
				item["name"] = key
				config, err := FlattenMap(value, "")
				if err != nil {
					return nil, fmt.Errorf("failed to flatten config for product %s: %w", key, err)
				}
				if _, ok := config["name"]; ok {
					return nil, fmt.Errorf("invalid product config: top-level key %q is reserved", "name")
				}
				maps.Copy(item, config)
			} else {
				return nil, fmt.Errorf("invalid config for products")
			}
			output = append(output, item)
		}
	default:
		return nil, fmt.Errorf("product configuration must be map[string]any")
	}
	return output, nil
}

func ConvertStringMapToAny(m map[string]string) map[string]any {
	result := make(map[string]any)
	for k, v := range m {
		result[k] = v
	}
	return result
}

func FlattenMapRecursive(input map[string]any, prefix string, output map[string]any) {
	for key, value := range input {
		newKey := key
		if prefix != "" {
			newKey = prefix + "." + newKey
		}
		switch v := value.(type) {
		case map[string]any:
			FlattenMapRecursive(v, newKey, output)
		case map[string]string:
			newMap := ConvertStringMapToAny(v)
			FlattenMapRecursive(newMap, newKey, output)
		default:
			output[newKey] = value
		}
	}
}

func FlattenMap(input any, prefix string) (map[string]interface{}, error) {
	output := make(map[string]interface{})
	switch config := input.(type) {
	case string:
		output[prefix] = config
	case map[string]any:
		FlattenMapRecursive(config, prefix, output)
	default:
		return nil, fmt.Errorf("not supported config format")
	}
	return output, nil
}

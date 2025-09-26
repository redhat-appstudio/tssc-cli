package config

import (
	"fmt"
)

// ExtractFromStringMap extract config path and values from string map
func ExtractFromStringMap(m map[string]interface{}, prefix string, keyPaths *[]string, values *[]interface{}) {
	for key, value := range m {
		currentPath := key
		if prefix != "" {
			currentPath = prefix + "." + key
		}
		switch v := value.(type) {
		case map[string]interface{}:
			ExtractFromStringMap(v, currentPath, keyPaths, values)
		case map[interface{}]interface{}:
			ExtractFromInterfaceMap(v, currentPath, keyPaths, values)
		default:
			*keyPaths = append(*keyPaths, currentPath)
			*values = append(*values, value)
		}
	}
}

// ExtractFromInterfaceMap extract path and value from interface map
func ExtractFromInterfaceMap(m map[interface{}]interface{}, prefix string, keyPaths *[]string, values *[]interface{}) {
	for k, value := range m {
		key := fmt.Sprintf("%v", k)
		currentPath := key
		if prefix != "" {
			currentPath = prefix + "." + key
		}
		switch v := value.(type) {
		case map[string]interface{}:
			ExtractFromStringMap(v, currentPath, keyPaths, values)
		case map[interface{}]interface{}:
			ExtractFromInterfaceMap(v, currentPath, keyPaths, values)
		default:
			*keyPaths = append(*keyPaths, currentPath)
			*values = append(*values, value)
		}
	}
}

// ExtractKeyPathsAndValues return keyPaths and values of new config
func ExtractKeyPathsAndValues(data interface{}, prefix string) ([]string, []interface{}) {
	var keyPaths []string
	var values []interface{}
	switch m := data.(type) {
	case map[string]interface{}:
		ExtractFromStringMap(m, prefix, &keyPaths, &values)
	case map[interface{}]interface{}:
		ExtractFromInterfaceMap(m, prefix, &keyPaths, &values)
	}
	return keyPaths, values
}

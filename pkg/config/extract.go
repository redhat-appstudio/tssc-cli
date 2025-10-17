package config

// ConvertStringMapToAny converts a map[string]string to a map[string]any.
func ConvertStringMapToAny(m map[string]string) map[string]any {
	result := make(map[string]any)
	for k, v := range m {
		result[k] = v
	}
	return result
}

// FlattenMapRecursive recursively flattens a nested map[string]any into a
// single-level map. Keys are concatenated with dots to represent their original
// hierarchy ("key path").
func FlattenMapRecursive(
	input map[string]any,
	prefix string,
	output map[string]any,
) {
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

// FlattenMap flattens a given input into a single-level map.  If the input is a
// map[string]any, it calls FlattenMapRecursive to flatten it, using the provided
// prefix for keys. If the input is not a map, it treats the entire input as a
// single value and assigns it to the given prefix.
func FlattenMap(input any, prefix string) (map[string]interface{}, error) {
	output := make(map[string]interface{})
	switch config := input.(type) {
	case map[string]any:
		FlattenMapRecursive(config, prefix, output)
	default:
		output[prefix] = input
	}
	return output, nil
}

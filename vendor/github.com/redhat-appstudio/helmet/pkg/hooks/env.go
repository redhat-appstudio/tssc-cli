package hooks

import (
	"fmt"
	"strings"
)

// valuesToEnv flattens a map of values into a map of environment variables using
// "__" as a separator to merge the original keys into a single variable.
func valuesToEnv(vals map[string]interface{}, parentKey string) map[string]string {
	flatMap := map[string]string{}
	for k, v := range vals {
		newKey := strings.ToUpper(k)
		if parentKey != "" {
			newKey = fmt.Sprintf(
				"%s__%s",
				strings.ToUpper(parentKey),
				strings.ToUpper(k),
			)
		}
		switch child := v.(type) {
		case map[string]interface{}:
			childMap := valuesToEnv(child, newKey)
			for k, v := range childMap {
				flatMap[k] = v
			}
		default:
			flatMap[newKey] = fmt.Sprintf("%v", v)
		}
	}
	return flatMap
}

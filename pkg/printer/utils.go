package printer

import (
	"fmt"
	"strings"
)

func valuesToProperties(
	vals map[string]interface{},
	path string,
	sb *strings.Builder,
) {
	for k, v := range vals {
		newPath := k
		if path != "" {
			newPath = path + "." + k
		}
		switch v := v.(type) {
		case map[string]interface{}:
			valuesToProperties(v, newPath, sb)
		default:
			sb.WriteString(fmt.Sprintf("%s: %v\n", newPath, v))
		}
	}
}

func printProperties(sb *strings.Builder, prefix string) {
	lines := strings.Split(sb.String(), "\n")
	for i, line := range lines {
		if i < len(lines)-1 {
			fmt.Printf("%s%s\n", prefix, line)
		}
	}
}

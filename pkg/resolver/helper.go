package resolver

import (
	"strings"
)

// commaSeparatedToSlice splits a comma-separated string into a slice of strings.
// It trims whitespace and skips empty parts.
func commaSeparatedToSlice(commaSeparated string) []string {
	// Removing all whitespace from the input string.
	commaSeparated = strings.TrimSpace(commaSeparated)
	if commaSeparated == "" {
		return nil
	}
	// Splitting the comma-separated string into individual parts.
	parts := strings.Split(commaSeparated, ",")
	slice := make([]string, 0, len(parts))
	for _, p := range parts {
		// Skipping any empty parts.
		if name := strings.TrimSpace(p); name != "" {
			slice = append(slice, name)
		}
	}
	return slice
}

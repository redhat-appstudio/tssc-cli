package resolver

import (
	"path/filepath"
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

// bundleIDFromChartPath returns the bundles/<id> directory name when chartPath is
// under bundles/<id>/charts/, otherwise "".
func bundleIDFromChartPath(chartPath string) string {
	chartPath = filepath.ToSlash(chartPath)
	parts := strings.Split(chartPath, "/")
	for i, p := range parts {
		if p == "bundles" && i+1 < len(parts) {
			return parts[i+1]
		}
	}
	return ""
}

func dedupeOrdered(parts []string) []string {
	seen := map[string]struct{}{}
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		if _, ok := seen[p]; ok {
			continue
		}
		seen[p] = struct{}{}
		out = append(out, p)
	}
	return out
}

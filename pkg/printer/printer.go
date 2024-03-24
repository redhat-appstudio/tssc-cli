package printer

import (
	"fmt"
	"strings"

	"helm.sh/helm/v3/pkg/release"
)

// HelmReleasePrinter prints the release information.
func HelmReleasePrinter(rel *release.Release) {
	fmt.Println("#")
	fmt.Printf("#       Chart: %s\n", rel.Chart.Metadata.Name)
	fmt.Printf("#     Version: %s\n", rel.Chart.Metadata.Version)
	fmt.Printf("#      Status: %s\n", rel.Info.Status.String())
	fmt.Printf("#   Namespace: %s\n", rel.Namespace)
	fmt.Printf("#    Revision: %d\n", rel.Version)
	fmt.Printf("#     Updated: %s\n", rel.Info.LastDeployed.String())
	fmt.Println("#")

	if rel.Info.Notes != "" {
		fmt.Printf("#\n# Notes\n#\n\n")
		fmt.Println(rel.Info.Notes)
	}
}

// HelmExtendedReleasePrinter prints the release information, including the
// manifest and hooks.
func HelmExtendedReleasePrinter(rel *release.Release) {
	ValuesPrinter("Config", rel.Config)

	fmt.Printf("#\n# Manifest\n#\n\n")
	fmt.Print(rel.Manifest)

	if len(rel.Hooks) > 0 {
		fmt.Printf("#\n# Hooks\n#\n")
		for _, hook := range rel.Hooks {
			fmt.Printf("---\n%s\n", hook.Manifest)
		}
	}
}

// ValuesPrinter prints the values in a map as properties.
func ValuesPrinter(title string, vals map[string]interface{}) {
	fmt.Printf("#\n# %s\n#\n\n", title)
	properties := new(strings.Builder)
	valuesToProperties(vals, "", properties)
	printProperties(properties, " * ")
}

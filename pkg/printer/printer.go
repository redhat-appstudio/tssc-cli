package printer

import (
	"fmt"
	"strings"

	"helm.sh/helm/v3/pkg/release"
)

// HelmReleasePrinter prints the release information.
func HelmReleasePrinter(rel *release.Release) {
	fmt.Printf("#\n#       Chart: %s\n", rel.Chart.Metadata.Name)
	fmt.Printf("#     Version: %s\n", rel.Chart.Metadata.Version)
	fmt.Printf("#      Status: %s\n", rel.Info.Status.String())
	fmt.Printf("#   Namespace: %s\n", rel.Namespace)
	fmt.Printf("#    Revision: %d\n", rel.Version)
	fmt.Printf("#     Updated: %s\n", rel.Info.LastDeployed.String())

	fmt.Printf("#\n# Notes:\n#\n")
	fmt.Println(rel.Info.Notes)
}

// HelmExtendedReleasePrinter prints the release information, including the
// manifest and hooks.
func HelmExtendedReleasePrinter(rel *release.Release) {
	ValuesPrinter(rel.Config)

	fmt.Printf("#\n# Manifest:\n#\n")
	fmt.Print(rel.Manifest)

	fmt.Printf("#\n# Hooks:\n#\n")
	for _, hook := range rel.Hooks {
		fmt.Printf("---\n%s\n", hook.Manifest)
	}
}

// ValuesPrinter prints the values in a map as properties.
func ValuesPrinter(vals map[string]interface{}) {
	fmt.Printf("#\n# Config:\n#\n")
	properties := new(strings.Builder)
	valuesToProperties(vals, "", properties)
	printProperties(properties, " * ")
}

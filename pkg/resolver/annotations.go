package resolver

import (
	"fmt"

	"github.com/redhat-appstudio/tssc-cli/pkg/constants"
)

var (
	// ProductNameAnnotation defines the product name a chart is reponsible for.
	ProductNameAnnotation = fmt.Sprintf("%s/product-name", constants.RepoURI)

	// DependsOnAnnotation defines the list of Helm chart names a chart requires
	// to be installed before it can be installed.
	DependsOnAnnotation = fmt.Sprintf("%s/depends-on", constants.RepoURI)

	// UseProductNamespaceAnnotation defines the Helm chart should use the same
	// namespace than the referred product name.
	UseProductNamespaceAnnotation = fmt.Sprintf(
		"%s/use-product-namespace",
		constants.RepoURI,
	)
)

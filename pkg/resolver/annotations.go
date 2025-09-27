package resolver

import (
	"fmt"

	"github.com/redhat-appstudio/tssc-cli/pkg/constants"
)

var (
	// ProductNameAnnotation defines the product name a chart is responsible for.
	ProductNameAnnotation = fmt.Sprintf("%s/product-name", constants.RepoURI)

	// DependsOnAnnotation defines the list of Helm chart names a chart requires
	// to be installed before it can be installed.
	DependsOnAnnotation = fmt.Sprintf("%s/depends-on", constants.RepoURI)

	// WeightAnnotation defines the weight a chart has in the topology, which
	// allows moving a dependency down or up in the topology. The annotation
	// "depends-on" takes preference over the weight annotation.
	WeightAnnotation = fmt.Sprintf("%s/weight", constants.RepoURI)

	// UseProductNamespaceAnnotation defines the Helm chart should use the same
	// namespace than the referred product name.
	UseProductNamespaceAnnotation = fmt.Sprintf(
		"%s/use-product-namespace",
		constants.RepoURI,
	)

	// IntegrationsProvidedAnnotation defines the list of integrations secrets a
	// Helm chart provides.
	IntegrationsProvidedAnnotation = fmt.Sprintf(
		"%s/integrations-provided",
		constants.RepoURI,
	)

	// IntegrationsRequiredAnnotation defines the list of integrations secrets a
	// Helm chart requires.
	IntegrationsRequiredAnnotation = fmt.Sprintf(
		"%s/integrations-required",
		constants.RepoURI,
	)
)

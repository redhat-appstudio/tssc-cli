package resolver

import (
	"fmt"

	"github.com/redhat-appstudio/tssc-cli/pkg/api"
)

// Annotation suffix constants
const (
	suffixProductName          = "product-name"
	suffixDependsOn            = "depends-on"
	suffixWeight               = "weight"
	suffixUseProductNamespace  = "use-product-namespace"
	suffixIntegrationsProvided = "integrations-provided"
	suffixIntegrationsRequired = "integrations-required"
)

// AnnotationProductName returns the product-name annotation key for the given app
// context.
func AnnotationProductName(appCtx *api.AppContext) string {
	return fmt.Sprintf("%s/%s", appCtx.RepoURI(), suffixProductName)
}

// AnnotationDependsOn returns the depends-on annotation key for the given app
// context.
func AnnotationDependsOn(appCtx *api.AppContext) string {
	return fmt.Sprintf("%s/%s", appCtx.RepoURI(), suffixDependsOn)
}

// AnnotationWeight returns the weight annotation key for the given app context.
func AnnotationWeight(appCtx *api.AppContext) string {
	return fmt.Sprintf("%s/%s", appCtx.RepoURI(), suffixWeight)
}

// AnnotationUseProductNamespace returns the use-product-namespace annotation key
// for the given app context.
func AnnotationUseProductNamespace(appCtx *api.AppContext) string {
	return fmt.Sprintf("%s/%s", appCtx.RepoURI(), suffixUseProductNamespace)
}

// AnnotationIntegrationsProvided returns the integrations-provided annotation key
// for the given app context.
func AnnotationIntegrationsProvided(appCtx *api.AppContext) string {
	return fmt.Sprintf("%s/%s", appCtx.RepoURI(), suffixIntegrationsProvided)
}

// AnnotationIntegrationsRequired returns the integrations-required annotation key
// for the given app context.
func AnnotationIntegrationsRequired(appCtx *api.AppContext) string {
	return fmt.Sprintf("%s/%s", appCtx.RepoURI(), suffixIntegrationsRequired)
}

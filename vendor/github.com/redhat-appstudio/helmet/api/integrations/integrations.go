package integrations

import "context"

// IntegrationContext provides cluster and installer configuration
// to a URLProvider without exposing internal types. Implementations
// are supplied by the framework when calling the provider.
type IntegrationContext interface {
	// GetOpenShiftIngressDomain returns the OpenShift ingress domain for the cluster.
	// Returns an error if the cluster is not OpenShift or the domain cannot be determined.
	GetOpenShiftIngressDomain(ctx context.Context) (string, error)
	// GetProductNamespace returns the namespace for the named product from installer config.
	// Returns an error if the product is not found.
	GetProductNamespace(productName string) (string, error)
}

// URLProvider supplies URLs (callback for authentication, homepage, webhook).
// Used with framework.WithURLProvider. Implementations receive an
// IntegrationContext to derive URLs from cluster/config without importing internal.
type URLProvider interface {
	GetCallbackURL(ctx context.Context, ic IntegrationContext) (string, error)
	GetHomepageURL(ctx context.Context, ic IntegrationContext) (string, error)
	GetWebhookURL(ctx context.Context, ic IntegrationContext) (string, error)
}

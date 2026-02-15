package api

import "github.com/redhat-appstudio/helmet/api/integrations"

// IntegrationContext provides cluster and installer configuration
// to a URLProvider without exposing internal types. Implementations
// are supplied by the framework when calling the provider.
//
// This is a re-export of integrations.IntegrationContext for convenience.
type IntegrationContext = integrations.IntegrationContext

// URLProvider supplies URLs (callback for authentication, homepage, webhook).
// Used with framework.WithURLProvider. Implementations receive an
// IntegrationContext to derive URLs from cluster/config without importing internal.
//
// This is a re-export of integrations.URLProvider for convenience.
type URLProvider = integrations.URLProvider

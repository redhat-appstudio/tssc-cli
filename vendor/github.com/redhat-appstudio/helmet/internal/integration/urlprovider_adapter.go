package integration

import (
	"context"

	"github.com/redhat-appstudio/helmet/api/integrations"
	"github.com/redhat-appstudio/helmet/internal/config"
	"github.com/redhat-appstudio/helmet/internal/k8s"
	"github.com/redhat-appstudio/helmet/internal/runcontext"
)

// urlProviderAdapter implements integration.URLProvider by delegating to
// integrations.URLProvider, and implements integrations.IntegrationContext from
// *runcontext.RunContext and *config.Config. Used when wiring a public
// URLProvider into the GitHub integration (e.g. in setClusterURLs).
type urlProviderAdapter struct {
	provider integrations.URLProvider
	runCtx   *runcontext.RunContext
	cfg      *config.Config
}

// Ensure urlProviderAdapter implements both interfaces at compile time.
var (
	_ integrations.IntegrationContext = (*urlProviderAdapter)(nil)
	_ URLProvider                     = (*urlProviderAdapter)(nil)
)

// GetOpenShiftIngressDomain implements integrations.IntegrationContext.
func (a *urlProviderAdapter) GetOpenShiftIngressDomain(ctx context.Context) (string, error) {
	return k8s.GetOpenShiftIngressDomain(ctx, a.runCtx.Kube)
}

// GetProductNamespace implements integrations.IntegrationContext.
func (a *urlProviderAdapter) GetProductNamespace(productName string) (string, error) {
	product, err := a.cfg.GetProduct(productName)
	if err != nil {
		return "", err
	}
	return product.GetNamespace(), nil
}

// GetCallbackURL implements URLProvider by delegating to the public provider.
func (a *urlProviderAdapter) GetCallbackURL(ctx context.Context, _ *runcontext.RunContext, _ *config.Config) (string, error) {
	return a.provider.GetCallbackURL(ctx, a)
}

// GetHomepageURL implements URLProvider by delegating to the public provider.
func (a *urlProviderAdapter) GetHomepageURL(ctx context.Context, _ *runcontext.RunContext, _ *config.Config) (string, error) {
	return a.provider.GetHomepageURL(ctx, a)
}

// GetWebhookURL implements URLProvider by delegating to the public provider.
func (a *urlProviderAdapter) GetWebhookURL(ctx context.Context, _ *runcontext.RunContext, _ *config.Config) (string, error) {
	return a.provider.GetWebhookURL(ctx, a)
}

// newURLProviderAdapter returns an adapter that implements URLProvider by
// delegating to the given integrations.URLProvider with an IntegrationContext
// backed by runCtx and cfg. runCtx and cfg must be non-nil when the provider
// calls IntegrationContext methods (e.g. GetOpenShiftIngressDomain, GetProductNamespace).
func newURLProviderAdapter(provider integrations.URLProvider, runCtx *runcontext.RunContext, cfg *config.Config) *urlProviderAdapter {
	return &urlProviderAdapter{provider: provider, runCtx: runCtx, cfg: cfg}
}

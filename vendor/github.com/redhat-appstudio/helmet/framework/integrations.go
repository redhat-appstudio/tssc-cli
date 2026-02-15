package framework

import (
	"log/slog"

	"github.com/redhat-appstudio/helmet/api"
	"github.com/redhat-appstudio/helmet/internal/integration"
	"github.com/redhat-appstudio/helmet/internal/integrations"
	"github.com/redhat-appstudio/helmet/internal/k8s"
	"github.com/redhat-appstudio/helmet/internal/runcontext"
	"github.com/redhat-appstudio/helmet/internal/subcmd"
)

// GitHubModuleWithURLProvider returns a GitHub integration module that uses the
// given URLProvider for webhook, homepage, and optional callback URLs when not
// set by flags. Use this in custom installers (e.g. helmet-ex) to supply URLs
// from config or environment without importing internal.
func GitHubModuleWithURLProvider(provider api.URLProvider) api.IntegrationModule {
	return api.IntegrationModule{
		Name: string(integrations.GitHub),
		Init: func(l *slog.Logger, _ k8s.Interface) integration.Interface {
			gh := integration.NewGitHub(l)
			gh.SetURLProvider(provider)
			return gh
		},
		Command: func(appCtx *api.AppContext, runCtx *runcontext.RunContext, i *integration.Integration) api.SubCommand {
			return subcmd.NewIntegrationGitHub(appCtx, runCtx, i)
		},
	}
}

// WithURLProvider returns a copy of modules with the GitHub integration
// replaced by one that uses the given URLProvider. Use after StandardIntegrations()
// to customize GitHub App URLs (e.g. from env or config) without changing other
// integrations.
func WithURLProvider(modules []api.IntegrationModule, provider api.URLProvider) []api.IntegrationModule {
	out := make([]api.IntegrationModule, 0, len(modules))
	for _, m := range modules {
		if m.Name == string(integrations.GitHub) {
			out = append(out, GitHubModuleWithURLProvider(provider))
		} else {
			out = append(out, m)
		}
	}
	return out
}

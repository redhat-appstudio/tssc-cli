package framework

import (
	"fmt"
	"log/slog"
	"slices"
	"strings"

	"github.com/redhat-appstudio/helmet/api"
	"github.com/redhat-appstudio/helmet/internal/integration"
	"github.com/redhat-appstudio/helmet/internal/integrations"
	"github.com/redhat-appstudio/helmet/internal/k8s"
	"github.com/redhat-appstudio/helmet/internal/runcontext"
	"github.com/redhat-appstudio/helmet/internal/subcmd"
)

// SelectIntegrations returns the subset of modules whose names appear in names,
// in first-seen name order after duplicates are removed. If names is empty, it
// returns modules unchanged (same slice reference, no reordering). It returns an
// error if any name is not present in modules.
func SelectIntegrations(modules []api.IntegrationModule, names ...string) ([]api.IntegrationModule, error) {
	if len(names) == 0 {
		return modules, nil
	}
	byName := make(map[string]api.IntegrationModule, len(modules))
	for _, m := range modules {
		byName[m.Name] = m
	}
	seen := make(map[string]struct{}, len(names))
	unique := make([]string, 0, len(names))
	for _, n := range names {
		if strings.TrimSpace(n) == "" {
			return nil, fmt.Errorf("integration name cannot be empty")
		}
		if _, ok := seen[n]; ok {
			continue
		}
		seen[n] = struct{}{}
		unique = append(unique, n)
	}
	var unknown []string
	for _, n := range unique {
		if _, ok := byName[n]; !ok {
			unknown = append(unknown, n)
		}
	}
	if len(unknown) > 0 {
		slices.Sort(unknown)
		return nil, fmt.Errorf("unknown integration names: %s", strings.Join(unknown, ", "))
	}
	out := make([]api.IntegrationModule, 0, len(unique))
	for _, n := range unique {
		out = append(out, byName[n])
	}
	return out, nil
}

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

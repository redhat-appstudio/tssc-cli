package subcmd

import (
	"log/slog"

	"github.com/redhat-appstudio/helmet/api"
	"github.com/redhat-appstudio/helmet/internal/integration"
	"github.com/redhat-appstudio/helmet/internal/integrations"
	"github.com/redhat-appstudio/helmet/internal/k8s"
	"github.com/redhat-appstudio/helmet/internal/runcontext"
)

var (
	ACSModule = api.IntegrationModule{
		Name: string(integrations.ACS),
		Init: func(_ *slog.Logger, _ *k8s.Kube) integration.Interface {
			return integration.NewACS()
		},
		Command: func(appCtx *api.AppContext, runCtx *runcontext.RunContext, i *integration.Integration) api.SubCommand {
			return NewIntegrationACS(appCtx, runCtx, i)
		},
	}

	ArtifactoryModule = api.IntegrationModule{
		Name: string(integrations.Artifactory),
		Init: func(_ *slog.Logger, _ *k8s.Kube) integration.Interface {
			return integration.NewContainerRegistry("")
		},
		Command: func(appCtx *api.AppContext, runCtx *runcontext.RunContext, i *integration.Integration) api.SubCommand {
			return NewIntegrationArtifactory(appCtx, runCtx, i)
		},
	}

	AzureModule = api.IntegrationModule{
		Name: string(integrations.Azure),
		Init: func(_ *slog.Logger, _ *k8s.Kube) integration.Interface {
			return integration.NewAzure()
		},
		Command: func(appCtx *api.AppContext, runCtx *runcontext.RunContext, i *integration.Integration) api.SubCommand {
			return NewIntegrationAzure(appCtx, runCtx, i)
		},
	}

	BitBucketModule = api.IntegrationModule{
		Name: string(integrations.BitBucket),
		Init: func(_ *slog.Logger, _ *k8s.Kube) integration.Interface {
			return integration.NewBitBucket()
		},
		Command: func(appCtx *api.AppContext, runCtx *runcontext.RunContext, i *integration.Integration) api.SubCommand {
			return NewIntegrationBitBucket(appCtx, runCtx, i)
		},
	}

	GitHubModule = api.IntegrationModule{
		Name: string(integrations.GitHub),
		Init: func(l *slog.Logger, k *k8s.Kube) integration.Interface {
			return integration.NewGitHub(l, k)
		},
		Command: func(appCtx *api.AppContext, runCtx *runcontext.RunContext, i *integration.Integration) api.SubCommand {
			return NewIntegrationGitHub(appCtx, runCtx, i)
		},
	}

	GitLabModule = api.IntegrationModule{
		Name: string(integrations.GitLab),
		Init: func(l *slog.Logger, _ *k8s.Kube) integration.Interface {
			return integration.NewGitLab(l)
		},
		Command: func(appCtx *api.AppContext, runCtx *runcontext.RunContext, i *integration.Integration) api.SubCommand {
			return NewIntegrationGitLab(appCtx, runCtx, i)
		},
	}

	JenkinsModule = api.IntegrationModule{
		Name: string(integrations.Jenkins),
		Init: func(_ *slog.Logger, _ *k8s.Kube) integration.Interface {
			return integration.NewJenkins()
		},
		Command: func(appCtx *api.AppContext, runCtx *runcontext.RunContext, i *integration.Integration) api.SubCommand {
			return NewIntegrationJenkins(appCtx, runCtx, i)
		},
	}

	NexusModule = api.IntegrationModule{
		Name: string(integrations.Nexus),
		Init: func(_ *slog.Logger, _ *k8s.Kube) integration.Interface {
			return integration.NewContainerRegistry("")
		},
		Command: func(appCtx *api.AppContext, runCtx *runcontext.RunContext, i *integration.Integration) api.SubCommand {
			return NewIntegrationNexus(appCtx, runCtx, i)
		},
	}

	QuayModule = api.IntegrationModule{
		Name: string(integrations.Quay),
		Init: func(_ *slog.Logger, _ *k8s.Kube) integration.Interface {
			return integration.NewContainerRegistry(integration.QuayURL)
		},
		Command: func(appCtx *api.AppContext, runCtx *runcontext.RunContext, i *integration.Integration) api.SubCommand {
			return NewIntegrationQuay(appCtx, runCtx, i)
		},
	}

	TrustedArtifactSignerModule = api.IntegrationModule{
		Name: string(integrations.TrustedArtifactSigner),
		Init: func(_ *slog.Logger, _ *k8s.Kube) integration.Interface {
			return integration.NewTrustedArtifactSigner()
		},
		Command: func(appCtx *api.AppContext, runCtx *runcontext.RunContext, i *integration.Integration) api.SubCommand {
			return NewIntegrationTrustedArtifactSigner(appCtx, runCtx, i)
		},
	}

	TrustificationModule = api.IntegrationModule{
		Name: string(integrations.Trustification),
		Init: func(_ *slog.Logger, _ *k8s.Kube) integration.Interface {
			return integration.NewTrustification()
		},
		Command: func(appCtx *api.AppContext, runCtx *runcontext.RunContext, i *integration.Integration) api.SubCommand {
			return NewIntegrationTrustification(appCtx, runCtx, i)
		},
	}
)

// StandardModules returns the list of standard integration modules.
func StandardModules() []api.IntegrationModule {
	return []api.IntegrationModule{
		ACSModule,
		ArtifactoryModule,
		AzureModule,
		BitBucketModule,
		GitHubModule,
		GitLabModule,
		JenkinsModule,
		NexusModule,
		QuayModule,
		TrustedArtifactSignerModule,
		TrustificationModule,
	}
}

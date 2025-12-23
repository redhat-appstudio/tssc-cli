package subcmd

import (
	"log/slog"

	"github.com/redhat-appstudio/tssc-cli/pkg/api"
	"github.com/redhat-appstudio/tssc-cli/pkg/integration"
	"github.com/redhat-appstudio/tssc-cli/pkg/integrations"
	"github.com/redhat-appstudio/tssc-cli/pkg/k8s"
)

var (
	ACSModule = api.IntegrationModule{
		Name: string(integrations.ACS),
		Init: func(_ *slog.Logger, _ *k8s.Kube) integration.Interface {
			return integration.NewACS()
		},
		Command: func(l *slog.Logger, k *k8s.Kube, i *integration.Integration) api.SubCommand {
			return NewIntegrationACS(l, k, i)
		},
	}

	ArtifactoryModule = api.IntegrationModule{
		Name: string(integrations.Artifactory),
		Init: func(_ *slog.Logger, _ *k8s.Kube) integration.Interface {
			return integration.NewContainerRegistry("")
		},
		Command: func(l *slog.Logger, k *k8s.Kube, i *integration.Integration) api.SubCommand {
			return NewIntegrationArtifactory(l, k, i)
		},
	}

	AzureModule = api.IntegrationModule{
		Name: string(integrations.Azure),
		Init: func(_ *slog.Logger, _ *k8s.Kube) integration.Interface {
			return integration.NewAzure()
		},
		Command: func(l *slog.Logger, k *k8s.Kube, i *integration.Integration) api.SubCommand {
			return NewIntegrationAzure(l, k, i)
		},
	}

	BitBucketModule = api.IntegrationModule{
		Name: string(integrations.BitBucket),
		Init: func(_ *slog.Logger, _ *k8s.Kube) integration.Interface {
			return integration.NewBitBucket()
		},
		Command: func(l *slog.Logger, k *k8s.Kube, i *integration.Integration) api.SubCommand {
			return NewIntegrationBitBucket(l, k, i)
		},
	}

	GitHubModule = api.IntegrationModule{
		Name: string(integrations.GitHub),
		Init: func(l *slog.Logger, k *k8s.Kube) integration.Interface {
			return integration.NewGitHub(l, k)
		},
		Command: func(l *slog.Logger, k *k8s.Kube, i *integration.Integration) api.SubCommand {
			return NewIntegrationGitHub(l, k, i)
		},
	}

	GitLabModule = api.IntegrationModule{
		Name: string(integrations.GitLab),
		Init: func(l *slog.Logger, _ *k8s.Kube) integration.Interface {
			return integration.NewGitLab(l)
		},
		Command: func(l *slog.Logger, k *k8s.Kube, i *integration.Integration) api.SubCommand {
			return NewIntegrationGitLab(l, k, i)
		},
	}

	JenkinsModule = api.IntegrationModule{
		Name: string(integrations.Jenkins),
		Init: func(_ *slog.Logger, _ *k8s.Kube) integration.Interface {
			return integration.NewJenkins()
		},
		Command: func(l *slog.Logger, k *k8s.Kube, i *integration.Integration) api.SubCommand {
			return NewIntegrationJenkins(l, k, i)
		},
	}

	NexusModule = api.IntegrationModule{
		Name: string(integrations.Nexus),
		Init: func(_ *slog.Logger, _ *k8s.Kube) integration.Interface {
			return integration.NewContainerRegistry("")
		},
		Command: func(l *slog.Logger, k *k8s.Kube, i *integration.Integration) api.SubCommand {
			return NewIntegrationNexus(l, k, i)
		},
	}

	QuayModule = api.IntegrationModule{
		Name: string(integrations.Quay),
		Init: func(_ *slog.Logger, _ *k8s.Kube) integration.Interface {
			return integration.NewContainerRegistry(integration.QuayURL)
		},
		Command: func(l *slog.Logger, k *k8s.Kube, i *integration.Integration) api.SubCommand {
			return NewIntegrationQuay(l, k, i)
		},
	}

	TrustedArtifactSignerModule = api.IntegrationModule{
		Name: string(integrations.TrustedArtifactSigner),
		Init: func(_ *slog.Logger, _ *k8s.Kube) integration.Interface {
			return integration.NewTrustedArtifactSigner()
		},
		Command: func(l *slog.Logger, k *k8s.Kube, i *integration.Integration) api.SubCommand {
			return NewIntegrationTrustedArtifactSigner(l, k, i)
		},
	}

	TrustificationModule = api.IntegrationModule{
		Name: string(integrations.Trustification),
		Init: func(_ *slog.Logger, _ *k8s.Kube) integration.Interface {
			return integration.NewTrustification()
		},
		Command: func(l *slog.Logger, k *k8s.Kube, i *integration.Integration) api.SubCommand {
			return NewIntegrationTrustification(l, k, i)
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

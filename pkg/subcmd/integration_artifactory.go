package subcmd

import (
	"log/slog"

	"github.com/redhat-appstudio/tssc/pkg/config"
	"github.com/redhat-appstudio/tssc/pkg/integrations"
	"github.com/redhat-appstudio/tssc/pkg/k8s"

	"github.com/spf13/cobra"
)

// IntegrationArtifactory is the sub-command for the "integration artifactory",
// responsible for creating and updating the Artifactory integration secret.
type IntegrationArtifactory struct {
	cmd    *cobra.Command // cobra command
	logger *slog.Logger   // application logger
	cfg    *config.Config // installer configuration
	kube   *k8s.Kube      // kubernetes client

	artifactoryIntegration *integrations.ArtifactoryIntegration // artifactory integration

	apiToken         string // web API token
	dockerconfigjson string // credentials to push/pull from the registry
}

var _ Interface = &IntegrationArtifactory{}

const artifactoryIntegrationLongDesc = `
Manages the artifactory integration with TSSC, by storing the required
credentials required by the TSSC services to interact with artifactory.

The credentials are stored in a Kubernetes Secret in the configured namespace
for RHDH.
`

// Cmd exposes the cobra instance.
func (d *IntegrationArtifactory) Cmd() *cobra.Command {
	return d.cmd
}

// Complete is a no-op in this case.
func (d *IntegrationArtifactory) Complete(args []string) error {
	var err error
	d.cfg, err = bootstrapConfig(d.cmd.Context(), d.kube)
	return err
}

// Validate checks if the required configuration is set.
func (d *IntegrationArtifactory) Validate() error {
	return d.artifactoryIntegration.Validate()
}

// Run creates or updates the Artifactory integration secret.
func (d *IntegrationArtifactory) Run() error {
	err := d.artifactoryIntegration.EnsureNamespace(d.cmd.Context(), d.cfg)
	if err != nil {
		return err
	}
	return d.artifactoryIntegration.Create(d.cmd.Context(), d.cfg)
}

// NewIntegrationArtifactory creates the sub-command for the "integration artifactory"
// responsible to manage the TSSC integrations with a Artifactory image registry.
func NewIntegrationArtifactory(
	logger *slog.Logger,
	kube *k8s.Kube,
) *IntegrationArtifactory {
	artifactoryIntegration := integrations.NewArtifactoryIntegration(logger, kube)

	d := &IntegrationArtifactory{
		cmd: &cobra.Command{
			Use:          "artifactory [flags]",
			Short:        "Integrates a Artifactory instance into TSSC",
			Long:         artifactoryIntegrationLongDesc,
			SilenceUsage: true,
		},

		logger: logger,
		kube:   kube,

		artifactoryIntegration: artifactoryIntegration,
	}

	p := d.cmd.PersistentFlags()
	artifactoryIntegration.PersistentFlags(p)
	return d
}

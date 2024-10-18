package subcmd

import (
	"fmt"
	"log/slog"

	"github.com/redhat-appstudio/rhtap-cli/pkg/config"
	"github.com/redhat-appstudio/rhtap-cli/pkg/integrations"
	"github.com/redhat-appstudio/rhtap-cli/pkg/k8s"

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
Manages the artifactory integration with RHTAP, by storing the required
credentials required by the RHTAP services to interact with artifactory.

The credentials are stored in a Kubernetes Secret in the configured namespace
for RHDH.
`

// Cmd exposes the cobra instance.
func (d *IntegrationArtifactory) Cmd() *cobra.Command {
	return d.cmd
}

// Complete is a no-op in this case.
func (d *IntegrationArtifactory) Complete(args []string) error {
	return nil
}

// Validate checks if the required configuration is set.
func (d *IntegrationArtifactory) Validate() error {
	feature, err := d.cfg.GetFeature(config.RedHatDeveloperHub)
	if err != nil {
		return err
	}
	if !feature.Enabled {
		return fmt.Errorf("Red Hat Developer Hub feature is not enabled")
	}
	return d.artifactoryIntegration.Validate()
}

// Run creates or updates the Artifactory integration secret.
func (d *IntegrationArtifactory) Run() error {
	if err := d.artifactoryIntegration.EnsureNamespace(d.cmd.Context()); err != nil {
		return err
	}
	return d.artifactoryIntegration.Create(d.cmd.Context())
}

// NewIntegrationArtifactory creates the sub-command for the "integration artifactory"
// responsible to manage the RHTAP integrations with a Artifactory image registry.
func NewIntegrationArtifactory(
	logger *slog.Logger,
	cfg *config.Config,
	kube *k8s.Kube,
) *IntegrationArtifactory {
	artifactoryIntegration := integrations.NewArtifactoryIntegration(logger, cfg, kube)

	d := &IntegrationArtifactory{
		cmd: &cobra.Command{
			Use:          "artifactory [flags]",
			Short:        "Integrates a Artifactory instance into RHTAP",
			Long:         artifactoryIntegrationLongDesc,
			SilenceUsage: true,
		},

		logger: logger,
		cfg:    cfg,
		kube:   kube,

		artifactoryIntegration: artifactoryIntegration,
	}

	p := d.cmd.PersistentFlags()
	artifactoryIntegration.PersistentFlags(p)
	return d
}

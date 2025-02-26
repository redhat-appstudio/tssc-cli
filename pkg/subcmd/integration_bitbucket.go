package subcmd

import (
	"fmt"
	"log/slog"

	"github.com/redhat-appstudio/rhtap-cli/pkg/config"
	"github.com/redhat-appstudio/rhtap-cli/pkg/integrations"
	"github.com/redhat-appstudio/rhtap-cli/pkg/k8s"

	"github.com/spf13/cobra"
)

// IntegrationBitBucket is the sub-command for the "integration bitbucket",
// responsible for creating and updating the BitBucket integration secret.
type IntegrationBitBucket struct {
	cmd    *cobra.Command // cobra command
	logger *slog.Logger   // application logger
	cfg    *config.Config // installer configuration
	kube   *k8s.Kube      // kubernetes client

	bitbucketIntegration *integrations.BitBucketIntegration // bitbucket integration

	host         string // E.g. 'bitbucket.org'
	clientId     string // Application client id
	clientSecret string // Application client secret
	token        string // API token
}

var _ Interface = &IntegrationBitBucket{}

const bitbucketIntegrationLongDesc = `
Manages the BitBucket integration with RHTAP, by storing the required
credentials required by the RHTAP services to interact with BitBucket.

The credentials are stored in a Kubernetes Secret in the configured namespace
for RHDH.
`

// Cmd exposes the cobra instance.
func (d *IntegrationBitBucket) Cmd() *cobra.Command {
	return d.cmd
}

// Complete is a no-op in this case.
func (d *IntegrationBitBucket) Complete(args []string) error {
	var err error
	d.cfg, err = bootstrapConfig(d.cmd.Context(), d.kube)
	return err
}

// Validate checks if the required configuration is set.
func (d *IntegrationBitBucket) Validate() error {
	feature, err := d.cfg.GetFeature(config.RedHatDeveloperHub)
	if err != nil {
		return err
	}
	if !feature.Enabled {
		return fmt.Errorf("Red Hat Developer Hub feature is not enabled")
	}
	return d.bitbucketIntegration.Validate()
}

// Run creates or updates the BitBucket integration secret.
func (d *IntegrationBitBucket) Run() error {
	err := d.bitbucketIntegration.EnsureNamespace(d.cmd.Context(), d.cfg)
	if err != nil {
		return err
	}
	return d.bitbucketIntegration.Create(d.cmd.Context(), d.cfg)
}

// NewIntegrationBitBucket creates the sub-command for the "integration bitbucket"
// responsible to manage the RHTAP integrations with the BitBucket service.
func NewIntegrationBitBucket(
	logger *slog.Logger,
	kube *k8s.Kube,
) *IntegrationBitBucket {
	bitbucketIntegration := integrations.NewBitBucketIntegration(logger, kube)

	d := &IntegrationBitBucket{
		cmd: &cobra.Command{
			Use:          "bitbucket [flags]",
			Short:        "Integrates a BitBucket instance into RHTAP",
			Long:         bitbucketIntegrationLongDesc,
			SilenceUsage: true,
		},

		logger: logger,
		kube:   kube,

		bitbucketIntegration: bitbucketIntegration,
	}

	p := d.cmd.PersistentFlags()
	bitbucketIntegration.PersistentFlags(p)
	return d
}

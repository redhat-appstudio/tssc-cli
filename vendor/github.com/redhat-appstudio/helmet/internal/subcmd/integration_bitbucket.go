package subcmd

import (
	"log/slog"

	"github.com/redhat-appstudio/helmet/api"

	"github.com/redhat-appstudio/helmet/internal/config"
	"github.com/redhat-appstudio/helmet/internal/integration"
	"github.com/redhat-appstudio/helmet/internal/k8s"

	"github.com/spf13/cobra"
)

// IntegrationBitBucket is the sub-command for the "integration bitbucket",
// responsible for creating and updating the BitBucket integration secret.
type IntegrationBitBucket struct {
	cmd         *cobra.Command           // cobra command
	appCtx      *api.AppContext          // application context
	logger      *slog.Logger             // application logger
	cfg         *config.Config           // installer configuration
	kube        *k8s.Kube                // kubernetes client
	integration *integration.Integration // integration instance
}

var _ api.SubCommand = &IntegrationBitBucket{}

const bitbucketIntegrationLongDesc = `
Manages the BitBucket integration with TSSC, by storing the required
credentials required by the TSSC services to interact with BitBucket.

The credentials are stored in a Kubernetes Secret in the configured namespace
for RHDH.
`

// Cmd exposes the cobra instance.
func (b *IntegrationBitBucket) Cmd() *cobra.Command {
	return b.cmd
}

// Complete is a no-op in this case.
func (b *IntegrationBitBucket) Complete(args []string) error {
	var err error
	b.cfg, err = bootstrapConfig(b.cmd.Context(), b.appCtx, b.kube)
	return err
}

// Validate checks if the required configuration is set.
func (b *IntegrationBitBucket) Validate() error {
	return b.integration.Validate()
}

// Run creates or updates the BitBucket integration secret.
func (b *IntegrationBitBucket) Run() error {
	return b.integration.Create(b.cmd.Context(), b.cfg)
}

// NewIntegrationBitBucket creates the sub-command for the "integration bitbucket"
// responsible to manage the TSSC integrations with the BitBucket service.
func NewIntegrationBitBucket(
	appCtx *api.AppContext,
	logger *slog.Logger,
	kube *k8s.Kube,
	i *integration.Integration,
) *IntegrationBitBucket {
	b := &IntegrationBitBucket{
		cmd: &cobra.Command{
			Use:          "bitbucket [flags]",
			Short:        "Integrates a BitBucket instance into TSSC",
			Long:         bitbucketIntegrationLongDesc,
			SilenceUsage: true,
		},

		appCtx:      appCtx,
		logger:      logger,
		kube:        kube,
		integration: i,
	}
	i.PersistentFlags(b.cmd)
	return b
}

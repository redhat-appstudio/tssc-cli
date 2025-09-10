package subcmd

import (
	"log/slog"

	"github.com/redhat-appstudio/tssc-cli/pkg/config"
	"github.com/redhat-appstudio/tssc-cli/pkg/integration"
	"github.com/redhat-appstudio/tssc-cli/pkg/k8s"
	"k8s.io/apimachinery/pkg/types"

	"github.com/spf13/cobra"
)

// IntegrationBitBucket is the sub-command for the "integration bitbucket",
// responsible for creating and updating the BitBucket integration secret.
type IntegrationBitBucket struct {
	cmd         *cobra.Command           // cobra command
	logger      *slog.Logger             // application logger
	cfg         *config.Config           // installer configuration
	kube        *k8s.Kube                // kubernetes client
	integration *integration.Integration // integration instance
}

var _ Interface = &IntegrationBitBucket{}

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
	b.cfg, err = bootstrapConfig(b.cmd.Context(), b.kube)
	return err
}

// Validate checks if the required configuration is set.
func (b *IntegrationBitBucket) Validate() error {
	return b.integration.Validate()
}

// Run creates or updates the BitBucket integration secret.
func (b *IntegrationBitBucket) Run() error {
	return b.integration.Create(b.cmd.Context(), b.cfg, types.NamespacedName{
		Namespace: b.cfg.Installer.Namespace,
		Name:      "tssc-bitbucket-integration",
	})
}

// NewIntegrationBitBucket creates the sub-command for the "integration bitbucket"
// responsible to manage the TSSC integrations with the BitBucket service.
func NewIntegrationBitBucket(
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

		logger:      logger,
		kube:        kube,
		integration: i,
	}
	i.PersistentFlags(b.cmd)
	return b
}

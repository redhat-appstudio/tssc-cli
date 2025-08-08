package subcmd

import (
	"context"
	"fmt"
	"os"

	"github.com/redhat-appstudio/tssc-cli/pkg/config"
	"github.com/redhat-appstudio/tssc-cli/pkg/constants"
	"github.com/redhat-appstudio/tssc-cli/pkg/k8s"
)

// bootstrapConfig helper to retrieve the cluster configuration.
func bootstrapConfig(
	ctx context.Context,
	kube *k8s.Kube,
) (*config.Config, error) {
	mgr := config.NewConfigMapManager(kube)
	cfg, err := mgr.GetConfig(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, `
Unable to find the configuration in the cluster, or the configuration is invalid.
Please refer to the subcommand "tssc config" to manage installer's
configuration for the target cluster.

	$ %s config --help
		`, constants.AppName)
	}
	return cfg, err
}

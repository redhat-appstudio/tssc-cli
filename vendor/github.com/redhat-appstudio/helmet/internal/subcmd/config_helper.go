package subcmd

import (
	"context"
	"fmt"
	"os"

	"github.com/redhat-appstudio/helmet/api"
	"github.com/redhat-appstudio/helmet/internal/config"
	"github.com/redhat-appstudio/helmet/internal/k8s"
)

// bootstrapConfig helper to retrieve the cluster configuration.
func bootstrapConfig(
	ctx context.Context,
	appCtx *api.AppContext,
	kube *k8s.Kube,
) (*config.Config, error) {
	mgr := config.NewConfigMapManager(kube, appCtx.Name)
	cfg, err := mgr.GetConfig(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, `
Unable to find the configuration in the cluster, or the configuration is invalid.
Please refer to the subcommand "%s config" to manage installer's
configuration for the target cluster.

	$ %s config --help
		`, appCtx.Name, appCtx.Name)
	}
	return cfg, err
}

package subcmd

import (
	"context"
	"fmt"
	"os"

	"github.com/redhat-appstudio/rhtap-cli/pkg/config"
	"github.com/redhat-appstudio/rhtap-cli/pkg/k8s"
)

func bootstrapConfig(
	ctx context.Context,
	kube *k8s.Kube,
) (*config.Config, error) {
	mgr := config.NewConfigMapManager(kube)
	cfg, err := mgr.GetConfig(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, `
Unable to find the configuration in the cluster, or the configuration is invalid.
Please refer to the subcommand "rhtap-cli config" to manage installer's
configuration for the target cluster.

	$ rhtap-cli config --help
		`)
	}
	return cfg, err
}

package subcmd

import (
	"log/slog"
	"os"

	"github.com/redhat-appstudio/tssc-cli/pkg/chartfs"
	"github.com/redhat-appstudio/tssc-cli/pkg/config"
	"github.com/redhat-appstudio/tssc-cli/pkg/k8s"
	"github.com/redhat-appstudio/tssc-cli/pkg/resolver"

	"github.com/spf13/cobra"
)

// Topology represents the topology subcommand, it reports the installer
// dependency topology based on the cluster configuration and Helm charts.
type Topology struct {
	cmd    *cobra.Command   // cobra command
	logger *slog.Logger     // application logger
	cfs    *chartfs.ChartFS // embedded filesystem
	kube   *k8s.Kube        // kubernetes client

	collection *resolver.Collection // chart collection
	cfg        *config.Config       // installer configuration
}

var _ Interface = &Topology{}

const topologyDesc = `
Report the dependency topology of the installer based on the cluster configuration
and Helm charts. It will output a table with the following columns: 

  - Index: the index of the chart in the dependency graph.
  - Dependency: the name of the Helm chart.
  - Namespace: the OpenShift namespace where the chart is installed.
  - Product: the name of the product that the chart is associated with.
  - Depends-On: comma-separated list of charts that this chart depends on.
`

// Cmd exposes the cobra instance.
func (t *Topology) Cmd() *cobra.Command {
	return t.cmd
}

// Complete instantiates the cluster configuration and charts.
func (t *Topology) Complete(_ []string) error {
	// Load all charts from the embedded filesystem, or from a local directory.
	charts, err := t.cfs.GetAllCharts()
	if err != nil {
		return err
	}
	// Create a new chart collection from the loaded charts.
	if t.collection, err = resolver.NewCollection(charts); err != nil {
		return err
	}
	// Load the installer configuration from the cluster.
	if t.cfg, err = bootstrapConfig(t.cmd.Context(), t.kube); err != nil {
		return err
	}
	return nil
}

// Validate validates the command.
func (t *Topology) Validate() error {
	return nil
}

// Run resolves the dependency graph.
func (t *Topology) Run() error {
	// Resolving the dependency topology based on the installer configuration and
	// Helm charts.
	r := resolver.NewResolver(t.cfg, t.collection, resolver.NewTopology())
	if err := r.Resolve(); err != nil {
		return err
	}
	// Printing the resolved dependency to the standard output.
	r.Print(os.Stdout)
	return nil
}

// NewTopology instantiates a new Topology subcommand.
func NewTopology(
	logger *slog.Logger, // application logger
	cfs *chartfs.ChartFS, // chart filesystem
	kube *k8s.Kube, // Kubernetes client
) *Topology {
	t := &Topology{
		cmd: &cobra.Command{
			Use:          "topology",
			Short:        "Shows the installer topology",
			Long:         topologyDesc,
			SilenceUsage: true,
		},
		logger: logger.WithGroup("topology"),
		cfs:    cfs,
		kube:   kube,
	}
	return t
}

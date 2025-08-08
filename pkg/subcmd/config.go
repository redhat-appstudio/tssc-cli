package subcmd

import (
	"fmt"
	"log/slog"

	"github.com/redhat-appstudio/tssc-cli/pkg/chartfs"
	"github.com/redhat-appstudio/tssc-cli/pkg/config"
	"github.com/redhat-appstudio/tssc-cli/pkg/flags"
	"github.com/redhat-appstudio/tssc-cli/pkg/k8s"
	"github.com/redhat-appstudio/tssc-cli/pkg/printer"
	"github.com/redhat-appstudio/tssc-cli/pkg/resolver"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
)

type Config struct {
	cmd    *cobra.Command   // cobra command
	logger *slog.Logger     // application logger
	flags  *flags.Flags     // global flags
	cfs    *chartfs.ChartFS // embedded filesystem
	kube   *k8s.Kube        // kubernetes client

	manager    *config.ConfigMapManager // cluster configuration manager
	configPath string                   // configuration file relative path

	create bool // create a new configuration
	force  bool // overrides existing configuration
	get    bool // show the current configuration
	delete bool // delete the current configuration
}

var _ Interface = &Config{}

const configDesc = `
Manages installer's cluster configuration.

It should only be used to for experimental deployments. Production
deployments are not supported.

Before "tssc deploy", you need to
create a cluster configuration, responsible to define all installation settings
for the whole Kubernetes cluster.

You can use the embedded executable configuration, or inform your own local
configuration file path to "--create". Use "--force" to update existing
configuration.

The "--create" flag reflects the creation of a new configuration while, "--force"
is meant to amend the cluster configuration and overwrite changes to installer's
defaults.

This subcommand ensures a single cluster configuration is applied, identified and
retrieved using a unique label selector.
`

// Cmd exposes the cobra instance.
func (c *Config) Cmd() *cobra.Command {
	return c.cmd
}

// log returns a decorated logger.
func (c *Config) log() *slog.Logger {
	return c.flags.LoggerWith(c.logger.With("config-path", c.configPath))
}

// PersistentFlags injects the sub-command flags.
func (c *Config) PersistentFlags(p *pflag.FlagSet) {
	p.BoolVarP(
		&c.create,
		"create",
		"c",
		false,
		"Create new cluster configuration",
	)
	p.BoolVarP(
		&c.force,
		"force",
		"f",
		false,
		"Update an existing cluster configuration",
	)
	p.BoolVarP(
		&c.get,
		"get",
		"g",
		false,
		"Show the current cluster configuration",
	)
	p.BoolVarP(
		&c.delete,
		"delete",
		"d",
		false,
		"Delete the current cluster configuration",
	)
}

// validateFlags validates the flags passed to the subcommand.
func (c *Config) validateFlags() error {
	if c.get && c.delete {
		return fmt.Errorf("cannot get and delete at the same time")
	}
	if !c.create && !c.force && !c.get && !c.delete {
		return fmt.Errorf("either create, get or delete must be set")
	}
	return nil
}

// Complete inspect the context to determine the path of the configuration file,
// or uses the embedded payload, makes sure the args are adequate.
func (c *Config) Complete(args []string) error {
	// It should return an error if more than a single argument is informed.
	if len(args) > 1 {
		return fmt.Errorf("unexpected arguments: %v", args)
	}
	// It should inform a configuration file only for apply and update flags.
	if (c.get || c.delete) && !c.create && len(args) > 0 {
		return fmt.Errorf(
			"configuration file is only permitted for --create flag")
	}
	// Storing the configuration file reference, when empty using the embedded
	// default configuration path.
	if len(args) == 1 {
		c.configPath = args[0]
		c.log().Debug("Using local configuration file")
	} else {
		c.configPath = config.DefaultRelativeConfigPath
		c.log().Debug("Using embedded configuration file, default settings.")
	}
	return nil
}

// Validate make sure all items are in place.
func (c *Config) Validate() error {
	if c.create && c.configPath == "" {
		return fmt.Errorf("configuration file is not informed")
	}
	if err := c.validateFlags(); err != nil {
		return err
	}
	return nil
}

// runCreate runs create action, makes sure a new configuration is applied in the
// cluster and update when using the --force flag.
func (c *Config) runCreate() error {
	printer.Disclaimer()

	c.log().Debug("Loading configuration from file")
	cfg, err := config.NewConfigFromFile(c.cfs, c.configPath)
	if err != nil {
		return err
	}

	// Ensuring the configuration is compabile with the Helm charts available for
	// the installer, product associated charts and dependencies are verified.
	c.log().Debug("Verifying installer Helm charts")
	charts, err := c.cfs.GetAllCharts()
	if err != nil {
		return err
	}
	collection, err := resolver.NewCollection(charts)
	if err != nil {
		return err
	}
	r := resolver.NewResolver(cfg, collection, resolver.NewTopology())
	if err = r.Resolve(); err != nil {
		return err
	}

	if c.flags.DryRun {
		c.log().Debug("[DRY-RUN] Only showing the configuration payload")
		fmt.Printf(
			"[DRY-RUN] Creating the ConfigMap %q/%q, with the label selector %q\n",
			cfg.Installer.Namespace,
			config.Name,
			fmt.Sprintf("%s=true", config.Label),
		)
		fmt.Print(cfg.String())
		return nil
	}

	c.log().Debug("Making sure the OpenShift project is created")
	if err = k8s.EnsureOpenShiftProject(
		c.cmd.Context(),
		c.log(),
		c.kube,
		cfg.Installer.Namespace,
	); err != nil {
		return err
	}

	c.log().Debug("Applying the new configuration in the cluster")
	if err = c.manager.Create(c.cmd.Context(), cfg); err != nil {
		if apierrors.IsAlreadyExists(err) {
			if c.force {
				c.log().Debug("Updating the configuration in the cluster")
				return c.manager.Update(c.cmd.Context(), cfg)
			} else {
				return fmt.Errorf(
					"the configuration already exists, use --force to amend it")
			}
		}
	}
	return err
}

// runDelete controls the deletion process.
func (c *Config) runDelete() error {
	if c.flags.DryRun {
		c.log().Debug("[DRY-RUN] Configuration is not removed from the cluster")
		fmt.Printf(
			"[DRY-RUN] Removing the ConfigMap %q, with the label selector %q\n",
			config.Name,
			fmt.Sprintf("%s=true", config.Label),
		)
		return nil
	}
	return c.manager.Delete(c.cmd.Context())
}

// runGet controls the cluster configuration retrieval process.
func (c *Config) runGet() error {
	c.log().Debug("Retrieving the cluster configuration")
	cfg, err := c.manager.GetConfig(c.cmd.Context())
	if err != nil {
		if c.create && c.flags.DryRun {
			c.log().Warn(
				"[DRY-RUN] Configuration does not exist in the cluster, yet.")
			return nil
		}
		return err
	}
	c.log().Debug("Formatting the configuration as string")
	fmt.Print(cfg.String())
	return nil
}

// Run runs the subcommand main action, checks which flags are enabled to interact
// with cluster's configuration.
func (c *Config) Run() error {
	var err error
	switch {
	case c.create:
		if err = c.runCreate(); err != nil {
			return err
		}
	case c.delete:
		if err = c.runDelete(); err != nil {
			return err
		}
	}

	// The --get flag can take place together with other flags, thus this block
	// evaluation takes place after the switch block.
	if c.get {
		if err = c.runGet(); err != nil {
			return err
		}
	}
	return nil
}

// NewConfig instantiates the "config" subcommand.
func NewConfig(
	logger *slog.Logger,
	f *flags.Flags,
	cfs *chartfs.ChartFS,
	kube *k8s.Kube,
) Interface {
	c := &Config{
		cmd: &cobra.Command{
			Use:          "config [flags] [path/to/config.yaml]",
			Short:        "Manages installer's cluster configuration",
			Long:         configDesc,
			SilenceUsage: true,
		},
		logger:  logger.WithGroup("config"),
		flags:   f,
		cfs:     cfs,
		kube:    kube,
		manager: config.NewConfigMapManager(kube),
	}

	c.PersistentFlags(c.cmd.PersistentFlags())

	return c
}

package subcmd

import (
	"fmt"
	"log/slog"

	"github.com/redhat-appstudio/rhtap-cli/pkg/config"
	"github.com/redhat-appstudio/rhtap-cli/pkg/flags"
	"github.com/redhat-appstudio/rhtap-cli/pkg/installer"
	"github.com/redhat-appstudio/rhtap-cli/pkg/k8s"

	"github.com/spf13/cobra"
)

// Template represents the "template" subcommand.
type Template struct {
	cmd    *cobra.Command // cobra command
	logger *slog.Logger   // application logger
	flags  *flags.Flags   // global flags
	cfg    *config.Config // installer configuration
	kube   *k8s.Kube      // kubernetes client

	// TODO: add support for "--validate", so the rendered resources are validated
	// against the cluster during templating.

	valuesTemplatePath string            // path to the values template file
	showValues         bool              // show rendered values
	showManifests      bool              // show rendered manifests
	dep                config.Dependency // chart to render
}

var _ Interface = &Template{}

const templateDesc = `
The Template subcommand is used to render the values template file and,
optionally, the Helm chart manifests. It is particularly useful for
troubleshooting and developing Helm charts for the RHTAP installation process.

By using the '--show-manifest=false' flag, only the values template
('--values-template') will be rendered, making the last argument, with the Helm
chart directory, optional.

Additionally, the '--debug' flag should be used to display the raw values template
payload, regardless of whether it is valid YAML or not.

The installer resources are embedded in the executable, these resources are
employed by default, to use local files, set the '--embedded' flag to false.
`

// Cmd exposes the cobra instance.
func (t *Template) Cmd() *cobra.Command {
	return t.cmd
}

// log logger with contextual information.
func (t *Template) log() *slog.Logger {
	return t.flags.LoggerWith(
		t.dep.LoggerWith(
			t.logger.With("values-template", t.valuesTemplatePath),
		),
	)
}

// Complete parse the informed args as charts, when valid.
func (t *Template) Complete(args []string) error {
	// Dry-run mode is always enabled by default for templating, when manually set
	// to false it will return a validation error.
	t.flags.DryRun = true

	if !t.showManifests {
		return nil
	}
	if len(args) != 1 {
		return fmt.Errorf("expecting one chart, got %d", len(args))
	}
	t.dep.Chart = args[0]
	return nil
}

// Validate checks if the chart path is a directory.
func (t *Template) Validate() error {
	if !t.showManifests {
		return nil
	}
	if !t.flags.DryRun {
		return fmt.Errorf("template command is only available in dry-run mode")
	}
	if t.dep.Chart == "" {
		return fmt.Errorf("missing chart path")
	}
	return nil
}

// Run Renders the templates.
func (t *Template) Run() error {
	cfs, err := newChartFS(t.logger, t.flags, t.cfg)
	if err != nil {
		return err
	}

	valuesTmplPayload, err := cfs.ReadFile(t.valuesTemplatePath)
	if err != nil {
		return fmt.Errorf("failed to read values template file: %w", err)
	}

	// Installer for the specific dependency
	i := installer.NewInstaller(t.logger, t.flags, t.kube, cfs, &t.dep)

	// Setting values and loading cluster's information.
	if err = i.SetValues(
		t.cmd.Context(),
		&t.cfg.Installer,
		string(valuesTmplPayload),
	); err != nil {
		return err
	}
	if t.showValues && t.flags.Debug {
		i.PrintRawValues()
	}

	// Rendering the global values.
	if err = i.RenderValues(); err != nil {
		return err
	}
	if t.showValues {
		i.PrintValues()
	}

	// When the manifests aren't shown, we don't need to dry-run "helm install".
	if !t.showManifests {
		return nil
	}
	return i.Install(t.cmd.Context())
}

// NewTemplate creates the "template" subcommand with flags.
func NewTemplate(
	logger *slog.Logger,
	f *flags.Flags,
	cfg *config.Config,
	kube *k8s.Kube,
) *Template {
	t := &Template{
		cmd: &cobra.Command{
			Use:          "template",
			Short:        "Render Helm chart templates",
			Long:         templateDesc,
			SilenceUsage: true,
		},
		logger:        logger.WithGroup("template"),
		flags:         f,
		cfg:           cfg,
		kube:          kube,
		dep:           config.Dependency{Namespace: "default"},
		showValues:    true,
		showManifests: true,
	}

	p := t.cmd.PersistentFlags()

	flags.SetValuesTmplFlag(p, &t.valuesTemplatePath)

	p.StringVar(&t.dep.Namespace, "namespace", t.dep.Namespace,
		"namespace to use on template rendering")
	p.BoolVar(&t.showValues, "show-values", t.showValues,
		"show values template rendered payload")
	p.BoolVar(&t.showManifests, "show-manifests", t.showManifests,
		"show Helm chart rendered manifests")

	return t
}

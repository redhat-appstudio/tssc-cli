package subcmd

import (
	"fmt"
	"log/slog"
	"os"

	"github.com/otaviof/rhtap-installer-cli/pkg/config"
	"github.com/otaviof/rhtap-installer-cli/pkg/deployer"
	"github.com/otaviof/rhtap-installer-cli/pkg/engine"
	"github.com/otaviof/rhtap-installer-cli/pkg/flags"
	"github.com/otaviof/rhtap-installer-cli/pkg/k8s"
	"github.com/otaviof/rhtap-installer-cli/pkg/printer"

	"github.com/spf13/cobra"
	"helm.sh/helm/v3/pkg/chartutil"
)

// Template represents the "template" subcommand.
type Template struct {
	cmd    *cobra.Command // cobra command
	logger *slog.Logger   // application logger
	flags  *flags.Flags   // global flags
	cfg    *config.Spec   // installer configuration
	kube   *k8s.Kube      // kubernetes client

	// TODO: add support for "--validate", so the rendered resources are validated
	// against the cluster during templating.

	valuesTemplatePath string            // path to the values template file
	showValues         bool              // show rendered values
	showManifests      bool              // show rendered manifests
	dependency         config.Dependency // chart to render
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
`

// Cmd exposes the cobra instance.
func (t *Template) Cmd() *cobra.Command {
	return t.cmd
}

// log logger with contextual information.
func (t *Template) log() *slog.Logger {
	return t.flags.LoggerWith(
		t.dependency.LoggerWith(
			t.logger.With("values-template", t.valuesTemplatePath),
		),
	)
}

// Complete parse the informed args as charts, when valid.
func (t *Template) Complete(args []string) error {
	// Dry-run mode is always enabled by default for templating, when manually set
	// fo false it will return a validation error.
	t.flags.DryRun = true

	if !t.showManifests {
		return nil
	}
	if len(args) != 1 {
		return fmt.Errorf("expecting one chart, got %d", len(args))
	}
	t.dependency.Chart = args[0]
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
	if t.dependency.Chart == "" {
		return fmt.Errorf("missing chart path")
	}
	info, err := os.Stat(t.dependency.Chart)
	if err != nil {
		return err
	}
	if !info.IsDir() {
		return fmt.Errorf("chart path %s is not a directory", t.dependency.Chart)
	}
	return nil
}

// Run Renders the templates.
func (t *Template) Run() error {
	t.log().Debug("Preparing values template context")
	variables := engine.NewVariables()
	if err := variables.SetInstaller(t.cfg); err != nil {
		return err
	}
	if err := variables.SetOpenShift(t.cmd.Context(), t.kube); err != nil {
		return err
	}

	t.log().Debug("Loading values template file")
	valuesTemplatePayload, err := os.ReadFile(t.valuesTemplatePath)
	if err != nil {
		return err
	}

	t.log().Debug("Rendering values from template")
	eng := engine.NewEngine(t.kube, string(valuesTemplatePayload))
	valuesBytes, err := eng.Render(variables)
	if err != nil {
		return err
	}

	if t.showValues && t.flags.Debug {
		t.log().Debug("Showing raw results of rendered values template")
		fmt.Printf("#\n# Values (Raw)\n#\n\n%s\n", valuesBytes)
	}

	t.log().Debug("Preparing Helm values")
	values, err := chartutil.ReadValues(valuesBytes)
	if err != nil {
		return err
	}

	if t.showValues {
		t.log().Debug("Showing parsed values")
		printer.ValuesPrinter("Values", values)
	}

	if !t.showManifests {
		return nil
	}

	t.log().Debug("Showing rendered chart manifests")
	hc, err := deployer.NewHelm(t.logger, t.flags, t.kube, t.dependency)
	if err != nil {
		return err
	}
	return hc.Install(values)
}

// NewTemplate creates the "template" subcommand with flags.
func NewTemplate(
	logger *slog.Logger,
	f *flags.Flags,
	cfg *config.Spec,
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
		dependency:    config.Dependency{Namespace: "default"},
		showValues:    true,
		showManifests: true,
	}

	p := t.cmd.PersistentFlags()

	flags.SetValuesTmplFlag(p, &t.valuesTemplatePath)

	p.StringVar(&t.dependency.Namespace, "namespace", t.dependency.Namespace,
		"namespace to use on template rendering")
	p.BoolVar(&t.showValues, "show-values", t.showValues,
		"show values template rendered payload")
	p.BoolVar(&t.showManifests, "show-manifests", t.showManifests,
		"show Helm chart rendered manifests")

	return t
}

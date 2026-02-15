package subcmd

import (
	"fmt"

	"github.com/redhat-appstudio/helmet/api"
	"github.com/redhat-appstudio/helmet/internal/config"
	"github.com/redhat-appstudio/helmet/internal/flags"
	"github.com/redhat-appstudio/helmet/internal/installer"
	"github.com/redhat-appstudio/helmet/internal/resolver"
	"github.com/redhat-appstudio/helmet/internal/runcontext"

	"github.com/spf13/cobra"
)

// Template represents the "template" subcommand.
type Template struct {
	cmd    *cobra.Command // cobra command
	appCtx *api.AppContext
	runCtx *runcontext.RunContext
	flags  *flags.Flags
	cfg    *config.Config // installer configuration

	valuesTemplatePath string              // path to the values template file
	showValues         bool                // show rendered values
	showManifests      bool                // show rendered manifests
	namespace          string              // dependency namespace
	dep                resolver.Dependency // chart to render
	installerTarball   []byte              // embedded installer tarball
}

var _ api.SubCommand = (*Template)(nil)

const templateDesc = `
The Template subcommand is used to render the values template file and,
optionally, the Helm chart manifests. It is particularly useful for
troubleshooting and developing Helm charts for the TSSC installation process.

By using the '--show-manifest=false' flag, only the global values template
('--values-template') will be rendered as YAML, thus the last argument, with the
Helm chart directory, optional.

Additionally, the '--debug' flag should be used to display rendered global values,
passed into every Helm Chart installed, as key-value pairs.

The installer resources are embedded in the executable, these resources are
employed by default, to use local files just use the last argument with the path
to the local Helm Chart.

Examples:

  # Only showing the global values as YAML.
  $ tssc template --show-manifests=false

  # Rendering only the templates of a single Helm Chart.
  $ tssc template --show-values=false charts/tssc-subscriptions

  # Rendering all resources of a Helm Chart.
  $ tssc template charts/tssc-subscriptions
`

// Cmd exposes the cobra instance.
func (t *Template) Cmd() *cobra.Command {
	return t.cmd
}

// Complete parse the informed args as charts, when valid.
func (t *Template) Complete(args []string) error {
	// Dry-run mode is always enabled by default for templating, when manually set
	// to false it will return a validation error.
	t.flags.DryRun = true

	if len(args) != 1 {
		return fmt.Errorf("expecting one chart, got %d", len(args))
	}

	hc, err := t.runCtx.ChartFS.GetChartFiles(args[0])
	if err != nil {
		return err
	}
	t.dep = *resolver.NewDependencyWithNamespace(hc, t.namespace)

	if t.cfg, err = bootstrapConfig(t.cmd.Context(), t.appCtx, t.runCtx); err != nil {
		return err
	}
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
	if t.dep.Chart() == nil {
		return fmt.Errorf("missing chart path")
	}
	return nil
}

// Run Renders the templates.
func (t *Template) Run() error {
	valuesTmplPayload, err := t.runCtx.ChartFS.ReadFile(t.valuesTemplatePath)
	if err != nil {
		return fmt.Errorf("failed to read values template file: %w", err)
	}

	i := installer.NewInstaller(t.runCtx.Logger, t.flags, t.runCtx.Kube, &t.dep, t.installerTarball)

	if err = i.SetValues(
		t.cmd.Context(),
		t.cfg,
		string(valuesTmplPayload),
	); err != nil {
		return err
	}

	// Rendering the global values.
	if err = i.RenderValues(); err != nil {
		return err
	}
	// Show the rendered global values, what's passed into very chart.
	if t.showValues {
		// Displaying the rendered values as properties, where it's easier to
		// verify settings by inspecting key-value pairs.
		// Show values as YAML.
		i.PrintRawValues()
	}

	// When the manifests aren't shown, we don't need to dry-run "helm install".
	if !t.showManifests {
		return nil
	}
	return i.Install(t.cmd.Context())
}

// NewTemplate creates the "template" subcommand with flags.
func NewTemplate(
	appCtx *api.AppContext,
	runCtx *runcontext.RunContext,
	f *flags.Flags,
	installerTarball []byte,
) *Template {
	t := &Template{
		cmd: &cobra.Command{
			Use:          "template",
			Short:        "Render Helm chart templates",
			Long:         templateDesc,
			SilenceUsage: true,
		},
		appCtx:           appCtx,
		runCtx:           runCtx,
		flags:            f,
		showValues:       true,
		showManifests:    true,
		namespace:        "default",
		installerTarball: installerTarball,
	}

	p := t.cmd.PersistentFlags()

	flags.SetValuesTmplFlag(p, &t.valuesTemplatePath)

	p.StringVar(&t.namespace, "namespace", t.namespace,
		"namespace to use on template rendering")
	p.BoolVar(&t.showValues, "show-values", t.showValues,
		"show values template rendered payload")
	p.BoolVar(&t.showManifests, "show-manifests", t.showManifests,
		"show Helm chart rendered manifests")

	return t
}

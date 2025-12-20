package flags

import (
	"github.com/redhat-appstudio/tssc-cli/pkg/api"

	"github.com/spf13/pflag"
)

// ValuesTemplateFlag flag name for the values template file.
const ValuesTemplateFlag = "values-template"

// SetValuesTmplFlag sets up the values-template flag to the informed pointer.
func SetValuesTmplFlag(p *pflag.FlagSet, v *string) {
	p.StringVar(
		v,
		ValuesTemplateFlag,
		api.ValuesFilename,
		"Path to the values template file",
	)
}

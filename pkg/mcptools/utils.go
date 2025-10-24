package mcptools

import (
	"fmt"
	"strings"

	"github.com/redhat-appstudio/tssc-cli/pkg/constants"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

// generateIntegrationSubCmdUsage generates a formatted usage string for an
// integration subcommand. It includes the command name, its long description, and
// an example usage showing required flags with placeholder values.
func generateIntegrationSubCmdUsage(cmd *cobra.Command) string {
	var usage strings.Builder
	usage.WriteString(fmt.Sprintf(
		"%s integration %s",
		constants.AppName, cmd.Name(),
	))

	cmd.PersistentFlags().VisitAll(func(f *pflag.Flag) {
		annotations, ok := f.Annotations[cobra.BashCompOneRequiredFlag]
		if ok && len(annotations) > 0 && annotations[0] == "true" {
			usage.WriteString(fmt.Sprintf(" --%s=\"OVERWRITE_ME\"", f.Name))
		}
	})

	return fmt.Sprintf(
		"## `%s` Subcommand Usage\n%s\nExample:\n\n\t%s\n",
		cmd.Name(), cmd.Long, usage.String())
}

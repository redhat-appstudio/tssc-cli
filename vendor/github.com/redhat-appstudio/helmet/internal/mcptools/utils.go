package mcptools

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

// missingClusterConfigErrorFromErr constructs a user-friendly error message
// indicating that the cluster configuration is missing. It advises the user
// on the next steps, including using `ConfigInitTool` to configure the cluster
// and `StatusToolName` to check the installation status.
func missingClusterConfigErrorFromErr(appName string, err error) string {
	return fmt.Sprintf(`
The cluster is not configured yet, use the tool %q to configure it. That's the
first step to deploy %s components. Next, you can use %q to check the overall
installation status.

Inspecting the configuration in the cluster returned the following error:

> %s`,
		appName+configInitSuffix,
		appName,
		appName+statusSuffix,
		err.Error(),
	)
}

// generateIntegrationSubCmdUsage generates a formatted usage string for an
// integration subcommand. It includes the command name, its long description, and
// an example usage showing required flags with placeholder values.
func generateIntegrationSubCmdUsage(appName string, cmd *cobra.Command) string {
	var usage strings.Builder
	usage.WriteString(fmt.Sprintf("%s integration %s", appName, cmd.Name()))

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

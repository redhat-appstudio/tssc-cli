package mcptools

import (
	"context"
	"fmt"
	"strings"

	"github.com/redhat-appstudio/tssc-cli/pkg/constants"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

type IntegrationTools struct {
	integrationCmd *cobra.Command // integration subcommand
}

const (
	// IntegrationListTool list integrations tool.
	IntegrationListTool = constants.AppName + "_integration_list"
	// IntegrationScaffoldTool generates the `tssc integration` command
	// required to configure the integration type
	IntegrationScaffoldTool = constants.AppName + "_integration_scaffold"
	// MandatoryIntegrations missing integrations that are mandatory
	MissingIntegration = "integration"
)

func (i *IntegrationTools) listHandler(
	ctx context.Context,
	ctr mcp.CallToolRequest,
) (*mcp.CallToolResult, error) {
	var output strings.Builder
	output.WriteString(fmt.Sprintf(`
# Integration Commands

The detailed description of each '%s integration' command is found below.
`,
		constants.AppName,
	))

	for _, subCmd := range i.integrationCmd.Commands() {
		var flagsInfo strings.Builder
		subCmd.PersistentFlags().VisitAll(func(f *pflag.Flag) {
			required := ""
			if _, value := f.Annotations[cobra.BashCompOneRequiredFlag]; value {
				if len(f.Annotations[cobra.BashCompOneRequiredFlag]) > 0 &&
					f.Annotations[cobra.BashCompOneRequiredFlag][0] == "true" {
					required = " (REQUIRED)"
				}
			}

			flagsInfo.WriteString(fmt.Sprintf(
				"  - \"--%s\" %s%s: %s.\n",
				f.Name,
				f.Value.Type(),
				required,
				f.Usage,
			))
		})
		output.WriteString(fmt.Sprintf(`
## '$ %s integration %s'

%s
%s

### Flags

%s
`,
			constants.AppName,
			subCmd.Name(),
			subCmd.Short,
			subCmd.Long,
			flagsInfo.String(),
		))
	}
	return mcp.NewToolResultText(output.String()), nil
}

// generateIntegrationCmd creates a command string for configuring an integration
func generateIntegrationCmd(subCmd *cobra.Command) string {
	var exampleCmd strings.Builder
	exampleCmd.WriteString(fmt.Sprintf("\n $ tssc integration %s \\\n", subCmd.Name()))

	var flags []string
	subCmd.PersistentFlags().VisitAll(func(f *pflag.Flag) {
		if annotations, ok := f.Annotations[cobra.BashCompOneRequiredFlag]; ok &&
			len(annotations) > 0 && annotations[0] == "true" {
			flags = append(flags, f.Name)
		}
	})

	for idx, flagName := range flags {
		if idx == len(flags)-1 {
			exampleCmd.WriteString(fmt.Sprintf(" --%s=\"REDACTED\"\n", flagName))
		} else {
			exampleCmd.WriteString(fmt.Sprintf(" --%s=\"REDACTED\" \\\n", flagName))
		}
	}

	return fmt.Sprintf(`
### Command to configure integration %s
%s:

%s
`, subCmd.Name(), subCmd.Short, exampleCmd.String())
}

func (i *IntegrationTools) scaffoldHandler(
	ctx context.Context,
	ctr mcp.CallToolRequest,
) (*mcp.CallToolResult, error) {
	var output strings.Builder
	var integration string

	output.WriteString(`
# Some integrations are missing, example commands will be provided.
  1. Copy and paste the commands in the terminal
  2. Replace "REDACTED" with your actual credentials
  3. Run the commands to configure the integrations
  4. Configure all the MANDATORY integrations
  5. Configure one of the OPTIONAL integrations
`)

	if i, ok := ctr.GetArguments()[MissingIntegration].(string); ok {
		integration = i
	}

	for _, subCmd := range i.integrationCmd.Commands() {
		if subCmd.Name() == integration {
			output.WriteString(generateIntegrationCmd(subCmd))
		}
	}

	return mcp.NewToolResultText(output.String()), nil
}

func (i *IntegrationTools) Init(s *server.MCPServer) {
	s.AddTools([]server.ServerTool{{
		Tool: mcp.NewTool(
			IntegrationListTool,
			mcp.WithDescription(`
List the TSSC integrations available for the user. Certain integrations are
required for certain features, make sure to configure the integrations
accordingly.`),
		),
		Handler: i.listHandler,
	},
		{
			Tool: mcp.NewTool(
				IntegrationScaffoldTool,
				mcp.WithDescription(`
Scaffold the configuration required for a specific TSSC integration. The
scaffolded configuration can be used as a reference to create the integration
using the 'tssc integration <name> ...' command.`),
				mcp.WithString(
					MissingIntegration,
					mcp.Description(`
The missing integrations that are mandatory for deployment.`,
					),
				),
			),
			Handler: i.scaffoldHandler,
		}}...)
}

func NewIntegrationTools(integrationCmd *cobra.Command) *IntegrationTools {
	return &IntegrationTools{
		integrationCmd: integrationCmd,
	}
}

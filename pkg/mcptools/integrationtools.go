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

func (i *IntegrationTools) Init(s *server.MCPServer) {
	s.AddTools([]server.ServerTool{{
		Tool: mcp.NewTool(
			"tssc_integration_list",
			mcp.WithDescription(`
List the TSSC integrations available for the user. Certain integrations are
required for certain features, make sure to configure the integrations
accordingly.`),
		),
		Handler: i.listHandler,
	}}...)
}

func NewIntegrationTools(integrationCmd *cobra.Command) *IntegrationTools {
	return &IntegrationTools{
		integrationCmd: integrationCmd,
	}
}

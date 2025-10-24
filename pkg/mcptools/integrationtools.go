package mcptools

import (
	"context"
	"fmt"
	"strings"

	"github.com/redhat-appstudio/tssc-cli/pkg/config"
	"github.com/redhat-appstudio/tssc-cli/pkg/constants"
	"github.com/redhat-appstudio/tssc-cli/pkg/integrations"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/spf13/cobra"
)

type IntegrationTools struct {
	integrationCmd *cobra.Command           // integration subcommand
	cm             *config.ConfigMapManager // configuration manager
	im             *integrations.Manager    // integrations manager
}

const (
	// IntegrationListTool list integrations tool.
	IntegrationListTool = constants.AppName + "_integration_list"
	// IntegrationScaffoldTool generates the `tssc integration` command
	// required to configure the integration type
	IntegrationScaffoldTool = constants.AppName + "_integration_scaffold"
	// IntegrationStatusTool checks if integrations are configured.
	IntegrationStatusTool = constants.AppName + "_integration_status"
)

// Arguments for the integration tools.
const (
	NamesArg = "names"
)

// listHandler generates a formatted string listing all available integration
// commands. It iterates through the registered subcommands of the integration
// command and appends their names and short descriptions to a string builder,
// which is then returned as a text tool result.
func (i *IntegrationTools) listHandler(
	ctx context.Context,
	ctr mcp.CallToolRequest,
) (*mcp.CallToolResult, error) {
	var output strings.Builder
	output.WriteString(fmt.Sprintf("# `%s` Integrations\n\n", constants.AppName))

	for _, subCmd := range i.integrationCmd.Commands() {
		output.WriteString(fmt.Sprintf("## `%s`\n\n%s\n\n",
			subCmd.Name(),
			subCmd.Short,
		))
	}
	return mcp.NewToolResultText(output.String()), nil
}

// scaffoldHandler generates scaffolded integration commands, explicitly warning
// that these handle sensitive information and must be manually executed by users,
// not automated agents.
func (i *IntegrationTools) scaffoldHandler(
	ctx context.Context,
	ctr mcp.CallToolRequest,
) (*mcp.CallToolResult, error) {
	var output strings.Builder

	output.WriteString(fmt.Sprintf(`# %s Integrations

Integrations handle sensitive information (e.g., passwords, tokens, credentials).
For security, automated agents **MUST NOT** execute these commands.

Users **MUST** manually copy and paste the example "%s integration" command, then
fill in the "OVERWRITE_ME" placeholders on a dedicated terminal session. For more
details, run "tssc integration <name> --help".`,
		constants.AppName, constants.AppName,
	))

	names := ctr.GetStringSlice(NamesArg, []string{})
	if len(names) == 0 {
		return mcp.NewToolResultErrorf(`
You must inform the %q argument, with the integration name(s)!`,
			NamesArg,
		), nil
	}

	byName := map[string]*cobra.Command{}
	for _, sc := range i.integrationCmd.Commands() {
		byName[sc.Name()] = sc
	}
	var unknown []string
	for _, name := range names {
		if sc, ok := byName[name]; ok {
			output.WriteString("\n\n")
			output.WriteString(generateIntegrationSubCmdUsage(sc))
			output.WriteString("\n\n")
		} else {
			unknown = append(unknown, name)
		}
	}
	if len(unknown) > 0 {
		return mcp.NewToolResultErrorf(
			"Unknown integration name(s): %s. Use %q to list valid names.",
			strings.Join(unknown, ", "), IntegrationListTool,
		), nil
	}

	return mcp.NewToolResultText(output.String()), nil
}

// integrationStatusHandler checks and reports the configuration status of
// specified integrations. It retrieves the cluster configuration and determines
// if each requested integration is configured.
func (i *IntegrationTools) integrationStatusHandler(
	ctx context.Context,
	ctr mcp.CallToolRequest,
) (*mcp.CallToolResult, error) {
	names := ctr.GetStringSlice(NamesArg, []string{})
	if len(names) == 0 {
		return mcp.NewToolResultErrorf(`
You must inform at least one integration name via the '%s' argument`,
			NamesArg,
		), nil
	}

	cfg, err := i.cm.GetConfig(ctx)
	if err != nil {
		return mcp.NewToolResultErrorFromErr(
			"Unable to load cluster configuration", err), nil
	}

	configured, err := i.im.ConfiguredIntegrations(ctx, cfg)
	if err != nil {
		return nil, err
	}

	configuredMap := make(map[string]bool)
	for _, c := range configured {
		configuredMap[c] = true
	}

	var output strings.Builder
	output.WriteString("# Integrations Status\n\n")

	for _, name := range names {
		if _, found := configuredMap[name]; found {
			output.WriteString(fmt.Sprintf("- `%s`: Configured\n", name))
		} else {
			output.WriteString(fmt.Sprintf("- `%s`: Not Configured\n", name))
		}
	}

	return mcp.NewToolResultText(output.String()), nil
}

// Init registers the TSSC integration management tools with the MCP server. These
// tools allow users to list available integrations, scaffold their
// configurations, and check their current status.
func (i *IntegrationTools) Init(s *server.MCPServer) {
	s.AddTools([]server.ServerTool{{
		Tool: mcp.NewTool(
			IntegrationListTool,
			mcp.WithDescription(`
List and describe the TSSC integrations available for the user.`),
		),
		Handler: i.listHandler,
	}, {
		Tool: mcp.NewTool(
			IntegrationScaffoldTool,
			mcp.WithDescription(`
Scaffold the configuration required for a specific TSSC integration. The
scaffolded configuration can be used as a reference to create the integration
using the 'tssc integration <name> ...' command.`,
			),
			mcp.WithArray(
				NamesArg,
				mcp.Description(`
The missing integrations that are mandatory for deployment.`,
				),
				mcp.WithStringItems(),
			),
		),
		Handler: i.scaffoldHandler,
	}, {
		Tool: mcp.NewTool(
			IntegrationStatusTool,
			mcp.WithDescription(`
Detect whether the informed integration names are configured.`,
			),
			mcp.WithArray(
				NamesArg,
				mcp.Description(`
The integration names to check the status for.`,
				),
				mcp.WithStringItems(),
			),
		),
		Handler: i.integrationStatusHandler,
	}}...)
}

func NewIntegrationTools(
	integrationCmd *cobra.Command,
	cm *config.ConfigMapManager,
	im *integrations.Manager,
) *IntegrationTools {
	return &IntegrationTools{
		integrationCmd: integrationCmd,
		cm:             cm,
		im:             im,
	}
}

package subcmd

import (
	"fmt"

	"github.com/redhat-appstudio/helmet/api"
	"github.com/redhat-appstudio/helmet/framework/mcpserver"
	"github.com/redhat-appstudio/helmet/internal/chartfs"
	"github.com/redhat-appstudio/helmet/internal/constants"
	"github.com/redhat-appstudio/helmet/internal/flags"
	"github.com/redhat-appstudio/helmet/internal/integrations"
	"github.com/redhat-appstudio/helmet/internal/k8s"
	"github.com/redhat-appstudio/helmet/internal/mcptools"

	"github.com/spf13/cobra"
)

// MCPServer is a subcommand for starting the MCP server.
type MCPServer struct {
	cmd    *cobra.Command   // cobra command
	appCtx *api.AppContext  // application context
	flags  *flags.Flags     // global flags
	cfs    *chartfs.ChartFS // embedded filesystem
	kube   *k8s.Kube        // kubernetes client

	manager         *integrations.Manager    // integrations manager
	mcpToolsBuilder mcptools.MCPToolsBuilder // builder function
	image           string                   // installer's container image
}

var _ api.SubCommand = &MCPServer{}

const mcpServerDesc = ` 
Starts the MCP server for the TSSC installer, using STDIO communication.
`

// PersistentFlags adds flags to the command.
func (m *MCPServer) PersistentFlags(cmd *cobra.Command) {
	p := cmd.PersistentFlags()
	p.StringVar(&m.image, "image", m.image, "container image for the installer\n")
}

// Cmd exposes the cobra instance.
func (m *MCPServer) Cmd() *cobra.Command {
	return m.cmd
}

// Complete implements api.SubCommand.
func (m *MCPServer) Complete(_ []string) error {
	return nil
}

// Validate implements api.SubCommand.
func (m *MCPServer) Validate() error {
	return nil
}

// Run starts the MCP server.
func (m *MCPServer) Run() error {
	// Create context using constructor - this ensures logger uses io.Discard
	toolsCtx := mcptools.NewMCPToolsContext(
		m.appCtx,
		m.flags,
		m.cfs,
		m.kube,
		m.manager,
		m.image,
	)

	// Invoke the builder to create tools
	tools, err := m.mcpToolsBuilder(toolsCtx)
	if err != nil {
		return fmt.Errorf("failed to create MCP tools: %w", err)
	}

	instructions, err := m.cfs.ReadFile(constants.InstructionsFilename)
	if err != nil {
		return fmt.Errorf("failed to read %s: %w",
			constants.InstructionsFilename, err)
	}

	s := mcpserver.NewMCPServer(m.appCtx, string(instructions))
	s.AddTools(tools...)

	return s.Start()
}

// NewMCPServer creates a new MCPServer instance.
func NewMCPServer(
	appCtx *api.AppContext,
	f *flags.Flags,
	cfs *chartfs.ChartFS,
	kube *k8s.Kube,
	manager *integrations.Manager,
	builder mcptools.MCPToolsBuilder,
	image string,
) *MCPServer {
	m := &MCPServer{
		cmd: &cobra.Command{
			Use:   "mcp-server",
			Short: "Starts the MCP server",
			Long:  mcpServerDesc,
		},

		appCtx:          appCtx,
		flags:           f,
		cfs:             cfs,
		kube:            kube,
		manager:         manager,
		mcpToolsBuilder: builder,
		image:           image,
	}

	m.PersistentFlags(m.cmd)
	return m
}

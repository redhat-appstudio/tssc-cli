package subcmd

import (
	"io"
	"log/slog"

	"github.com/redhat-appstudio/tssc-cli/pkg/config"
	"github.com/redhat-appstudio/tssc-cli/pkg/flags"
	"github.com/redhat-appstudio/tssc-cli/pkg/installer"
	"github.com/redhat-appstudio/tssc-cli/pkg/k8s"
	"github.com/redhat-appstudio/tssc-cli/pkg/mcpserver"
	"github.com/redhat-appstudio/tssc-cli/pkg/mcptools"

	"github.com/spf13/cobra"
)

// MCPServer is a subcommand for starting the MCP server.
type MCPServer struct {
	cmd    *cobra.Command // cobra command
	logger *slog.Logger   // application logger
	flags  *flags.Flags   // global flags
	kube   *k8s.Kube      // kubernetes client

	image string // installer's container image
}

var _ Interface = &MCPServer{}

const mcpServerDesc = ` 
Starts the MCP server for the TSSC installer, using STDIO communication.
`

// PersistentFlags adds flags to the command.
func (m *MCPServer) PersistentFlags(cmd *cobra.Command) {
	p := cmd.PersistentFlags()
	p.StringVar(&m.image, "image", "", "container image for the installer")

	if err := cmd.MarkPersistentFlagRequired("image"); err != nil {
		panic(err)
	}
}

// Cmd exposes the cobra instance.
func (m *MCPServer) Cmd() *cobra.Command {
	return m.cmd
}

// Complete implements Interface.
func (m *MCPServer) Complete(_ []string) error {
	return nil
}

// Validate implements Interface.
func (m *MCPServer) Validate() error {
	return nil
}

// Run starts the MCP server.
func (m *MCPServer) Run() error {
	cm := config.NewConfigMapManager(m.kube)
	cfgTools, err := mcptools.NewConfigTools(m.logger, m.kube, cm)
	if err != nil {
		return err
	}

	integrationCmd := NewIntegration(m.logger, m.kube)
	integrationTools := mcptools.NewIntegrationTools(integrationCmd)
	deployTools := mcptools.NewDeployTools(cm, installer.NewJob(m.kube), m.image)

	s := mcpserver.NewMCPServer()
	s.AddTools(cfgTools, integrationTools, deployTools)
	return s.Start()
}

// NewMCPServer creates a new MCPServer instance.
func NewMCPServer(f *flags.Flags, kube *k8s.Kube) *MCPServer {
	m := &MCPServer{
		cmd: &cobra.Command{
			Use:   "mcp-server",
			Short: "Starts the MCP server",
			Long:  mcpServerDesc,
		},
		// Given the MCP server runs via STDIO, we can't use the logger to output
		// to the console, for the time being it will be discarded.
		logger: f.GetLogger(io.Discard),
		flags:  f,
		kube:   kube,
	}
	m.PersistentFlags(m.cmd)
	return m
}

package subcmd

import (
	"fmt"
	"io"
	"log/slog"

	"github.com/redhat-appstudio/tssc-cli/pkg/api"
	"github.com/redhat-appstudio/tssc-cli/pkg/chartfs"
	"github.com/redhat-appstudio/tssc-cli/pkg/config"
	"github.com/redhat-appstudio/tssc-cli/pkg/constants"
	"github.com/redhat-appstudio/tssc-cli/pkg/flags"
	"github.com/redhat-appstudio/tssc-cli/pkg/installer"
	"github.com/redhat-appstudio/tssc-cli/pkg/integrations"
	"github.com/redhat-appstudio/tssc-cli/pkg/k8s"
	"github.com/redhat-appstudio/tssc-cli/pkg/mcpserver"
	"github.com/redhat-appstudio/tssc-cli/pkg/mcptools"
	"github.com/redhat-appstudio/tssc-cli/pkg/resolver"

	"github.com/spf13/cobra"
)

// MCPServer is a subcommand for starting the MCP server.
type MCPServer struct {
	cmd    *cobra.Command   // cobra command
	logger *slog.Logger     // application logger
	flags  *flags.Flags     // global flags
	cfs    *chartfs.ChartFS // embedded filesystem
	kube   *k8s.Kube        // kubernetes client

	manager *integrations.Manager // integrations manager
	image   string                // installer's container image
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
	cm := config.NewConfigMapManager(m.kube)
	configTools, err := mcptools.NewConfigTools(m.logger, m.cfs, m.kube, cm)
	if err != nil {
		return err
	}

	tb, err := resolver.NewTopologyBuilder(m.logger, m.cfs, m.manager)
	if err != nil {
		return err
	}
	jm := installer.NewJob(m.kube)
	statusTool := mcptools.NewStatusTool(cm, tb, jm)

	integrationCmd := NewIntegration(
		constants.AppName, m.logger, m.kube, m.cfs, m.manager,
	)
	integrationTools := mcptools.NewIntegrationTools(integrationCmd, cm, m.manager)

	deployTools := mcptools.NewDeployTools(cm, tb, jm, m.image)

	notesTool := mcptools.NewNotesTool(m.logger, m.flags, m.kube, cm, tb)

	topologyTool := mcptools.NewTopologyTool(m.cfs, cm, tb)

	instructions, err := m.cfs.ReadFile(constants.InstructionsFilename)
	if err != nil {
		return fmt.Errorf("failed to read %s: %w", constants.InstructionsFilename, err)
	}

	s := mcpserver.NewMCPServer(string(instructions))
	s.AddTools(configTools, statusTool, integrationTools, deployTools, notesTool, topologyTool)
	return s.Start()
}

// NewMCPServer creates a new MCPServer instance.
func NewMCPServer(
	f *flags.Flags,
	cfs *chartfs.ChartFS,
	kube *k8s.Kube,
	manager *integrations.Manager,
) *MCPServer {
	m := &MCPServer{
		cmd: &cobra.Command{
			Use:   "mcp-server",
			Short: "Starts the MCP server",
			Long:  mcpServerDesc,
		},
		// Given the MCP server runs via STDIO, we can't use the logger to output
		// to the console, for the time being it will be discarded.
		logger:  f.GetLogger(io.Discard),
		flags:   f,
		cfs:     cfs,
		kube:    kube,
		manager: manager,
	}

	m.image = "quay.io/redhat-user-workloads/rhtap-shared-team-tenant/tssc-cli"
	// Set default image based on CommitID
	if constants.CommitID == "" {
		m.image = fmt.Sprintf("%s:latest", m.image)
	} else {
		m.image = fmt.Sprintf("%s:%s", m.image, constants.CommitID)
	}
	m.PersistentFlags(m.cmd)
	return m
}

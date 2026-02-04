package mcpserver

import (
	"github.com/redhat-appstudio/helmet/api"
	"github.com/redhat-appstudio/helmet/internal/mcptools"

	"github.com/mark3labs/mcp-go/server"
)

type MCPServer struct {
	s *server.MCPServer // mcp server instance
}

func (m *MCPServer) AddTools(tools ...mcptools.Interface) {
	for _, tool := range tools {
		tool.Init(m.s)
	}
}

func (m *MCPServer) Start() error {
	return server.ServeStdio(m.s)
}

func NewMCPServer(appCtx *api.AppContext, instructions string) *MCPServer {
	return &MCPServer{s: server.NewMCPServer(
		appCtx.Name,
		appCtx.Version,
		server.WithToolCapabilities(true),
		server.WithPromptCapabilities(true),
		server.WithLogging(),
		server.WithInstructions(instructions),
	)}
}

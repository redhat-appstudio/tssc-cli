package mcptools

import (
	"github.com/mark3labs/mcp-go/server"
)

// Interface represents the MCP tools in the project.
type Interface interface {
	// Init decorates the MCP server with the tool declaration.
	Init(*server.MCPServer)
}

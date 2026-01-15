# Model Context Protocol (MCP) Server

The framework includes built-in MCP support, enabling AI assistants to interact with your installer programmatically.

## Usage

Configure your MCP client to run:

```sh
<installer-name> mcp --image="<container-image>"
```

The `--image` flag specifies the container image for Kubernetes Job-based deployments.

### Client Configuration

**Cursor** (`.cursor/mcp.json`):
```json
{
  "mcpServers": {
    "myapp": {
      "command": "myapp",
      "args": ["mcp", "--image=ghcr.io/myorg/myapp:latest"]
    }
  }
}
```

**Claude Desktop**:
```json
{
  "mcpServers": {
    "myapp": {
      "command": "/path/to/myapp",
      "args": ["mcp", "--image=ghcr.io/myorg/myapp:latest"]
    }
  }
}
```

## How It Works

- **Instructions**: Reads `instructions.md` from your installer filesystem to provide context to the AI
- **Tool Naming**: Automatically prefixes tools with your app name (e.g., `myapp_config_get`)
- **Long Operations**: Delegates deployments to Kubernetes Jobs to keep the server responsive
- **Communication**: Uses STDIO following the MCP specification

## Built-in MCP Tools

All tools are prefixed with `<app-name>_`. For example, if your app is `myapp`:

### Configuration

**`myapp_config_get`**
- Returns current or default configuration
- Arguments: None

**`myapp_config_create`**
- Creates new configuration
- Arguments: `namespace` (string, optional), `settings` (object, optional)

**`myapp_config_set`**
- Updates configuration values
- Arguments: `path` (string), `value` (any)

**`myapp_config_products_list`**
- Lists available products and their status
- Arguments: None

### Integrations

**`myapp_integration_list`**
- Lists available and configured integrations
- Arguments: None

**Note**: The MCP server does not configure integrations directly (they contain credentials). It provides instructions for manual CLI commands.

### Deployment

**`myapp_deploy`**
- Triggers deployment via Kubernetes Job
- Arguments: `dry_run` (boolean, optional), `namespace` (string, optional)

**`myapp_deploy_status`**
- Reports Job status and logs
- Arguments: None

### Topology

**`myapp_topology_get`**
- Returns dependency topology with installation order
- Arguments: None

### Status

**`myapp_status`**
- Comprehensive status report
- Arguments: None

## Custom Tools

Register custom tools when creating your app:

```go
import (
    "github.com/redhat-appstudio/helmet/pkg/api"
    "github.com/redhat-appstudio/helmet/pkg/framework/mcpserver"
)

func customTools(ctx api.AppContext, server *mcpserver.Server) error {
    server.AddTool(mcpserver.Tool{
        Name:        "myapp_backup",
        Description: "Creates a backup",
        InputSchema: map[string]interface{}{
            "type": "object",
            "properties": map[string]interface{}{
                "destination": {"type": "string", "description": "Backup path"},
            },
            "required": []string{"destination"},
        },
        Handler: func(args map[string]interface{}) (interface{}, error) {
            // Implementation
            return map[string]string{"status": "complete"}, nil
        },
    })
    return nil
}

func main() {
    app := framework.NewApp("myapp", filesystem,
        framework.WithMCPToolsBuilder(customTools),
    )
    app.Run()
}
```

## Example AI Prompts

**Initial Deployment**:
```
You are a Kubernetes platform engineer using the [your product] installer.
Help me deploy [products] to this cluster. Check the configuration and
guide me through any necessary integrations.
```

**Configuration Review**:
```
Show me the current configuration and explain which products are enabled.
List any missing integrations.
```

**Deployment**:
```
Show me the dependency topology, then deploy all enabled products.
```

## Best Practices

- **Clear Instructions**: Provide comprehensive `instructions.md` explaining your products and workflow
- **Semantic Names**: Use action-oriented tool names (e.g., `myapp_backup_create`)
- **Long Operations**: Delegate to Jobs for operations >30 seconds
- **Validate Inputs**: Always validate and return helpful error messages
- **Security**: Never pass credentials through MCP tools; use manual integration commands

## Security

- **Credentials**: Never expose credentials via MCP; provide CLI instructions instead
- **Authorization**: MCP server uses the user's `kubectl` permissions
- **Images**: Use specific tags, trusted registries, and consider image signing

## Troubleshooting

**MCP server won't start**: Check binary is in PATH, `instructions.md` exists, `kubectl` access works

**Tools not appearing**: Verify client config syntax, command path, server starts without errors

**Deployment Job fails**: Check `kubectl logs job/<installer-name>-deploy`, Job status, RBAC permissions

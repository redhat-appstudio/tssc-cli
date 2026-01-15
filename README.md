<p align="center">
    <a alt="Project quality report" href="https://goreportcard.com/report/github.com/redhat-appstudio/helmet">
        <img src="https://goreportcard.com/badge/github.com/redhat-appstudio/helmet">
    </a>
    <a alt="Release workflow status" href="https://github.com/redhat-appstudio/helmet/actions">
        <img src="https://github.com/redhat-appstudio/helmet/actions/workflows/release.yaml/badge.svg">
    </a>
    <a alt="Latest project release" href="https://github.com/redhat-appstudio/helmet/releases/latest">
        <img src="https://img.shields.io/github/v/release/redhat-appstudio/helmet">
    </a>
</p>

# Helmet

**A framework for building Kubernetes installers with Helm**

Helmet provides a reusable, batteries-included framework for creating intelligent Kubernetes installers that understand dependency relationships, manage configuration, and orchestrate complex deployments.

## Overview

Helmet is designed to be imported as a Go library to build custom installers for Kubernetes-based product suites. It handles the complexity of dependency resolution, configuration management, and deployment orchestration, allowing you to focus on defining your products and their relationships.

### Key Capabilities

- **Automatic Dependency Resolution**: Declares dependencies via Helm chart annotations and automatically determines installation order
- **Configuration Management**: YAML-based configuration with Kubernetes ConfigMap persistence
- **Template Engine**: Go templates for dynamic Helm values with cluster introspection
- **Integration System**: Pluggable integrations for external services (Git providers, registries, etc.)
- **Hook Scripts**: Execute custom logic before and after chart installations
- **CLI Generation**: Automatically generates a complete CLI with config, deploy, and integration commands
- **MCP Support**: Built-in Model Context Protocol server for AI assistant integration
- **Monitoring**: Resource readiness checks and Helm test execution

## Quick Start

```go
package main

import (
    _ "embed"
    "github.com/redhat-appstudio/helmet/pkg/framework"
)

//go:embed installer.tar
var installerAssets []byte

func main() {
    // Create filesystem from embedded assets
    filesystem, _ := framework.NewTarFS(installerAssets)

    // Create and run the installer
    app := framework.NewApp("myapp", filesystem,
        framework.WithVersion("1.0.0"),
    )

    app.Run()
}
```

This generates a complete CLI with commands for configuration, deployment, topology inspection, and more.

## Architecture

Helmet uses a convention-based approach where your installer assets are organized in a predictable structure:

```
installer/
├── config.yaml         # Configuration schema
├── values.yaml.tpl     # Go template for Helm values
├── charts/             # Helm charts directory
│   ├── database/
│   ├── api-server/
│   └── frontend/
└── instructions.md     # MCP server instructions (optional)
```

Charts declare dependencies and metadata via annotations:

```yaml
# charts/api-server/Chart.yaml
annotations:
  helmet.redhat-appstudio.github.com/product-name: "api"
  helmet.redhat-appstudio.github.com/depends-on: "database"
  helmet.redhat-appstudio.github.com/weight: "100"
```

The framework resolves dependencies, builds an installation topology, and orchestrates deployment in the correct order.

## Features

### Dependency Resolution

Automatically resolves and orders chart installations based on:
- Declared dependencies (`depends-on` annotation)
- Weight-based prioritization (`weight` annotation)
- Product associations (`product-name` annotation)
- Integration requirements (`integrations-required` CEL expressions)

See [Topology Documentation](docs/topology.md) for details.

### Configuration System

Product-based configuration with dynamic updates:

```yaml
myapp:
  settings:
    crc: false
  products:
    - name: test-product-name
      enabled: true
      namespace: test-product-namespace
      properties:
        manageSubscription: true
```

Configuration is stored as Kubernetes ConfigMaps and can be updated programmatically.

### Template Engine

Render Helm values dynamically based on configuration and cluster state:

```yaml
{{- $db := .Installer.Products.Database -}}
{{- if $db.Enabled }}
database:
  replicas: {{ $db.Properties.replicas | default 1 }}
  host: db.{{ .OpenShift.Ingress.Domain }}
{{- end }}
```

### Generated CLI

The framework automatically generates a complete CLI:

```bash
myapp config --create                    # Create configuration
myapp integration github --token=<token> # Configure integrations
myapp topology                           # View installation order
myapp deploy                            # Deploy all products
myapp mcp                               # Start MCP server
```

### Extensibility

Extend the framework with custom integrations, commands, and MCP tools:

```go
app := framework.NewApp("myapp", filesystem,
    framework.WithIntegrations(customIntegrations...),
    framework.WithMCPToolsBuilder(customMCPTools),
)

// Add custom commands
rootCmd := app.Command()
rootCmd.AddCommand(myCustomCommand)

app.Run()
```

## Installation

### As a Library

Import Helmet into your Go project:

```bash
go get github.com/redhat-appstudio/helmet/pkg/framework
```

## Documentation

- [Dependency Topology](docs/topology.md) - Chart dependency resolution and installation ordering
- [MCP Server](docs/mcp.md) - AI assistant integration via Model Context Protocol
- [Contributing Guide](CONTRIBUTING.md) - Development guidelines and best practices

## Examples

### Basic Installer

```go
package main

import (
    _ "embed"
    "github.com/redhat-appstudio/helmet/pkg/framework"
)

//go:embed installer.tar
var assets []byte

func main() {
    fs, _ := framework.NewTarFS(assets)
    app := framework.NewApp("myinstaller", fs)
    app.Run()
}
```

### With Custom Integrations

```go
import (
    "github.com/redhat-appstudio/helmet/pkg/api"
    "github.com/redhat-appstudio/helmet/pkg/framework"
)

type CustomIntegration struct{}

func (i *CustomIntegration) Name() string { return "custom" }
func (i *CustomIntegration) Init(ctx context.Context, logger logr.Logger, k8s *kubernetes.Clientset) (api.Integration, error) {
    // Implementation
    return &CustomIntegrationImpl{}, nil
}
func (i *CustomIntegration) Command(logger logr.Logger) (*cobra.Command, error) {
    // CLI command definition
    return &cobra.Command{Use: "custom"}, nil
}

func main() {
    fs, _ := framework.NewTarFS(assets)
    app := framework.NewApp("myinstaller", fs,
        framework.WithIntegrations(&CustomIntegration{}),
    )
    app.Run()
}
```

### With Custom MCP Tools

```go
import "github.com/redhat-appstudio/helmet/pkg/framework/mcpserver"

func customTools(ctx api.AppContext, s *mcpserver.Server) error {
    s.AddTool(mcpserver.Tool{
        Name:        "myapp_status",
        Description: "Get deployment status",
        Handler: func(args map[string]interface{}) (interface{}, error) {
            // Implementation
            return map[string]string{"status": "healthy"}, nil
        },
    })
    return nil
}

func main() {
    fs, _ := framework.NewTarFS(assets)
    app := framework.NewApp("myinstaller", fs,
        framework.WithMCPToolsBuilder(customTools),
    )
    app.Run()
}
```

## Design Principles

- **Convention over Configuration**: Predictable structure with sensible defaults
- **Interface-Driven**: Extensibility through well-defined interfaces
- **API Stability**: Functional options pattern for backward compatibility
- **Kubernetes-Native**: First-class support for namespaces, RBAC, and cluster introspection
- **Helm-Centric**: Leverages Helm's ecosystem and tooling

## Contributing

Contributions are welcome! Please see [CONTRIBUTING.md](CONTRIBUTING.md) for:

- Development environment setup
- Code style guidelines
- Testing requirements
- Pull request process

When contributing, please consider:
- API stability and backward compatibility
- Documentation for framework consumers
- Test coverage for new features
- Impact on existing consumers

## Resources

- [Project Homepage](https://github.com/redhat-appstudio/helmet)
- [Documentation](docs/)
- [Issue Tracker](https://github.com/redhat-appstudio/helmet/issues)
- [Releases](https://github.com/redhat-appstudio/helmet/releases)

---

**Note**: This is a framework for building installers. For specific product installers built with Helmet, please refer to their respective repositories.

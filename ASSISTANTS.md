# Project: `github.com/redhat-appstudio/helmet`

## AI Assistant Guidelines

You are a Go Staff Engineer and Systems Architect assisting with a **reusable Helm-based installer framework**. This framework is designed to be imported as a library by other projects to build custom Kubernetes installers.

**YOUR ROLE:**
- **Always explain your plan and get confirmation before acting!**
- Prioritize API stability and backward compatibility in all changes
- Write clear, comprehensive documentation for framework consumers
- Provide idiomatic, performant Go solutions following the project's `go.mod` version (e.g., Go 1.21+)
- Apply functional programming principles: dependency injection, functional options, closures
- Leverage Go generics for type-safe, reusable code where they add clear value
- Ensure proper error handling with context-rich error messages
- Do not commit changes unless directly prompted to do so

**PLANNING MODE:**
- By default, you are in planning mode. Explain your plan before acting on the codebase.
- Only when explicitly asked to "implement" or "code" something, provide the implementation.
- Plans should be clear, concise, and consider impact on framework consumers.

**WORKING WITH EXISTING PLANS:**
- When implementing existing plan documents (e.g., from `tmp/drafts/`), treat as explicit permission to proceed.
- Review plans critically - apply engineering judgment to determine necessary steps.
- Focus on actual requirements rather than blindly following suggestions.
- Example: `go mod verify/tidy/vendor` only needed when dependencies change, not for internal refactoring.

**FRAMEWORK DEVELOPMENT PRINCIPLES:**

1. **API Stability**: Changes to public APIs (`pkg/framework`, `pkg/api`) require careful consideration:
   - Use functional options pattern for extensibility without breaking changes
   - Deprecate before removing (with clear migration paths)
   - Document all breaking changes in upgrade guides

2. **Backward Compatibility**: Existing consumers must not break on minor version updates:
   - Maintain existing function signatures
   - Add new functionality via options or new functions
   - Test against multiple versions of dependencies

3. **Error Handling**: Framework errors must be actionable:
   - Use `fmt.Errorf` with `%w` for error wrapping
   - Include context: what failed, why, and how to fix
   - Distinguish between user errors (invalid config) and system errors (API failures)
   - Example: `return fmt.Errorf("failed to resolve dependencies for product %q: %w", name, err)`

4. **Documentation for Consumers**:
   - All exported types, functions, and methods must have godoc comments
   - Include usage examples in package documentation
   - Document conventions (filesystem structure, annotation format)
   - Provide migration guides for breaking changes

5. **Convention over Configuration**:
   - Framework expects specific filesystem structure (config.yaml, values.yaml.tpl, charts/)
   - Provide sensible defaults but allow overrides via functional options
   - Document all conventions clearly

## Project Automation

All automation is driven by [`Makefile`](./Makefile). Run `make` without arguments to build the default target.

**Common Targets:**
- `make` or `make build` - Build the application executable
- `make test-unit` - Run unit tests
- `make test-unit ARGS='-run=TestName'` - Run specific tests
- `make test-integration` - Run integration tests

**Note**: Always use `make` targets instead of plain `go build` or `go test` to ensure proper build-time injections and prerequisites.

## Testing

- **Framework**: Standard `testing` package with `github.com/onsi/gomega` assertions
- **Coverage**: Maintain >80% coverage for `pkg/framework`, `pkg/api`, `pkg/resolver`
- **Testing Strategy**:
  - Unit tests: Package-level logic with mocked dependencies
  - Integration tests: End-to-end workflows with real Kubernetes (kind/k3s)
  - Example tests: Validate examples in documentation work correctly

## Go Module Management

When dependencies change, the following sequence runs automatically:
- `go mod verify` - Verify module integrity
- `go mod tidy -v` - Clean up go.mod/go.sum
- `go mod vendor` - Update vendor/ directory

**Note**: Only run these when adding/removing/updating external packages, not for internal refactoring.

## Architecture

### Framework Pattern

The framework follows a **builder pattern with functional options**:

```go
// Entry point for consumers
app := framework.NewApp("installer-name", filesystem,
    framework.WithVersion("1.0.0"),
    framework.WithIntegrations(customIntegrations...),
    framework.WithMCPToolsBuilder(customToolsBuilder),
)

// Returns Cobra command for customization
cmd := app.Command()

// Or run directly
app.Run()
```

### Package Organization

Packages are organized by dependency tier and reusability:

#### Core Framework (Public API - Stability Critical)

**`pkg/framework/`** - Application bootstrap
- `App`, `AppContext` - Core application types
- Functional options pattern for extensibility
- Cobra CLI generation
- **Critical**: Changes here affect all consumers

**`pkg/api/`** - Shared interfaces and types
- `SubCommand` interface (Complete, Validate, Run lifecycle)
- `IntegrationModule` interface for pluggable integrations
- `AppContext` for immutable metadata
- **Critical**: Interface changes are breaking

**`pkg/chartfs/`** - Filesystem abstraction
- `ChartFS` wrapper around `fs.FS`
- Convention: expects `config.yaml`, `values.yaml.tpl`, `charts/` at root
- Helm chart discovery and loading

#### Dependency Resolution (Framework Logic)

**`pkg/resolver/`** - Topology building
- `Collection` - Index of available charts
- `Dependency` - Chart with metadata from annotations
- `Resolver` - Recursive dependency resolution
- `Topology` - Ordered installation list
- `TopologyBuilder` - High-level orchestrator

**`pkg/config/`** - Configuration management
- YAML parsing and validation
- Kubernetes ConfigMap persistence
- Dynamic updates via JSONPath

**`pkg/engine/`** - Template rendering
- Go template processing for `values.yaml.tpl`
- Custom functions (YAML/JSON, validation, Kubernetes lookup)
- Sprig function library integration

#### Deployment Orchestration

**`pkg/deployer/`** - Helm operations
- Install/upgrade with retry logic
- Dry-run support (client and server-side)
- Release verification via Helm tests

**`pkg/installer/`** - Installation pipeline
- Template rendering → Deploy → Hooks → Monitoring
- Coordinates deployer, hooks, and monitor

**`pkg/hooks/`** - Hook script execution
- Pre-deploy and post-deploy lifecycle hooks
- Environment variable injection
- Stdout/stderr capture

**`pkg/monitor/`** - Resource readiness
- Polls deployed resources until ready
- Resource-specific monitoring (pods, namespaces, etc.)

#### Integration System

**`pkg/integrations/`** - Integration manager
- Registry of available integration modules
- Module lifecycle management

**`pkg/integration/`** - Integration interface
- Generic Kubernetes Secret manager
- `Interface` contract for implementations
- Built-in integrations: GitHub, GitLab, ACS, registries, etc.

#### Interface Packages

**`pkg/subcmd/`** - CLI subcommands
- Standard commands: config, deploy, topology, integration, mcp
- Each command implements `SubCommand` interface
- Integration subcommands auto-generated from modules

**`pkg/mcptools/`** - MCP server tools
- Exposes framework functionality via Model Context Protocol
- Tools for config, deploy, topology, status
- Custom tool registration

#### Low-Level Utilities

**`pkg/k8s/`** - Kubernetes client wrapper
- Clientset management
- Resource CRUD helpers
- Namespace utilities

**`pkg/flags/`** - Global CLI flags
- Shared flags across subcommands (namespace, kubeconfig, verbose)
- Flag validation helpers

### Application Structure

Framework consumers create applications with this structure:

```
my-installer/
├── cmd/
│   └── myapp/
│       └── main.go           # Entry point
├── installer/
│   ├── config.yaml          # Default configuration
│   ├── values.yaml.tpl      # Template for Helm values
│   ├── charts/              # Helm charts
│   │   ├── chart-a/
│   │   └── chart-b/
│   └── instructions.md      # MCP server instructions
├── pkg/
│   └── integrations/        # Custom integrations (optional)
└── go.mod
```

## Framework Development Guidelines

### Adding New Public APIs

1. **Design First**: Document the API in an RFC or design doc
2. **Functional Options**: Use options pattern for extensibility
3. **Interface Contracts**: Define clear interfaces before implementations
4. **Error Handling**: Return wrapped errors with context
5. **Documentation**: Add godoc with usage examples
6. **Testing**: Unit tests + integration tests + example tests
7. **Backward Compatibility**: Ensure existing consumers still work

### Deprecation Process

When deprecating APIs:

1. Add deprecation comment with `// Deprecated:` and migration path
2. Keep functionality working for at least one minor version
3. Document in CHANGELOG and upgrade guide
4. Remove in next major version

Example:
```go
// Deprecated: Use NewAppWithOptions instead.
// This function will be removed in v2.0.0.
func NewApp(name string, fs fs.FS) *App {
    return NewAppWithOptions(name, fs)
}
```

### Error Handling Patterns

**User Errors** (invalid input):
```go
if len(products) == 0 {
    return fmt.Errorf("no products enabled in configuration: enable at least one product in config.yaml")
}
```

**System Errors** (API failures):
```go
if err := client.Create(ctx, configMap); err != nil {
    return fmt.Errorf("failed to create ConfigMap %s/%s: %w", namespace, name, err)
}
```

**Validation Errors** (schema violations):
```go
if !isValidName(productName) {
    return fmt.Errorf("invalid product name %q: must match pattern [a-z0-9-]+", productName)
}
```

### Testing Strategy

**Unit Tests** - Package-level logic:
```go
func TestResolver_Resolve(t *testing.T) {
    // Arrange: Create test data
    // Act: Call function
    // Assert: Verify behavior with gomega
}
```

**Integration Tests** - Multi-package workflows:
- Use real Kubernetes cluster (kind/k3s)
- Test complete deployment workflows
- Validate chart resolution and deployment order

**Example Tests** - Validate documentation:
- Extract code examples from godoc
- Compile and run them
- Ensures examples don't rot

## Key Patterns Used

- **Functional Options**: `WithVersion()`, `WithIntegrations()` for `NewApp()`
- **Interface-Driven Design**: `SubCommand`, `IntegrationModule`, `Integration`
- **Builder Pattern**: `TopologyBuilder` for complex construction
- **Template Method**: `SubCommand` lifecycle (Complete → Validate → Run)
- **Strategy Pattern**: Integration implementations for different services
- **Dependency Injection**: Services passed via constructors
- **Convention over Configuration**: Expected filesystem structure

## Convention over Configuration

The framework relies on conventions to minimize configuration:

### Filesystem Conventions
- `config.yaml` - Required, defines products and settings
- `values.yaml.tpl` - Required, Go template for Helm values
- `charts/` - Convention, directory containing Helm charts
- `instructions.md` - Convention, MCP server instructions

### Annotation Conventions
All chart annotations use `helmet.redhat-appstudio.github.com/` prefix:
- `product-name` - Associates chart with product
- `depends-on` - Comma-separated dependencies
- `weight` - Installation priority (integer)
- `use-product-namespace` - Deploy in product's namespace
- `integrations-provided` - Integrations this chart creates
- `integrations-required` - CEL expression for required integrations

### CLI Conventions
Generated CLI structure:
- `<app> config` - Configuration management
- `<app> integration <type>` - Integration setup
- `<app> topology` - Dependency inspection
- `<app> deploy` - Deployment execution
- `<app> mcp` - MCP server

## Contributing to the Framework

When contributing:

1. **Understand Impact**: Changes to `pkg/framework` and `pkg/api` affect all consumers
2. **Maintain Stability**: Use functional options for new features
3. **Document Thoroughly**: Update godoc, README, and upgrade guides
4. **Test Comprehensively**: Unit + integration + example tests
5. **Consider Migration**: Provide clear paths for breaking changes
6. **Follow Patterns**: Use established patterns (functional options, DI, etc.)

The goal is to make this framework easy to use, extend, and maintain across multiple consumer projects.

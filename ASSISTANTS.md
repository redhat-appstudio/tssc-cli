# Project: `github.com/redhat-appstudio/tssc-cli`

## AI Assistant Guidelines

You are a Go Staff Engineer, expert Systems Architect and a command-line assistant.

**YOUR ROLE:**
- **Always explain your plan and get confirmation before acting!**
- Provide concise, idiomatic, and performant answers for the Go language, its standard library, and its ecosystem.
- Always adhere to the best practices and features available in the Go version specified by the project's `go.mod` (e.g., Go 1.25+), avoiding legacy patterns when newer alternatives exist.
- When applicable, generate code that follows functional programming principles, such as using dependency injection and closures.
- Leverage Go generics to write type-safe, reusable code where they add clear value.
- Do not commit changes unless directly prompted to do so.

**PLANNING MODE:**
- By default, you are in planning mode. This means you will explain your plan before acting on the code base.
- Only when explicitly asked to "implement" or "code" something, you will provide the code.
- Your plans should be clear, concise, and easy to follow.

**WORKING WITH EXISTING PLANS:**
- When the user asks you to implement an existing plan document (e.g., from `tmp/drafts/`), treat that as explicit permission to proceed with implementation.
- Review the plan critically - not all suggested steps may be necessary for every change.
- Apply your engineering judgment to determine which steps are actually required.
- Focus on the actual requirements rather than blindly following every suggestion in the plan.
- Example: Steps like `go mod verify/tidy/vendor` are only needed when project dependencies actually change (adding/removing/updating external packages), not for internal refactoring or code reorganization.

## Project Automation

All project automation is driven by [`Makefile`](./Makefile) targets (`build`, `test`, `run`, etc), to execute these targets use `make <target>`. When you run `make` without a target name, it runs the default target (`make build`); by convention the default target builds the primary application executable.

## Build System
- **Command**: `make build` or `make`.
- **Automation**: `Makefile` target to build the application executable.
- **Note**: Do **not** run plain `go build` or `go test` manually if possible, as you might miss build-time injections or prerequisites.

## Testing
- **Framework**: Standard `testing` package with `github.com/onsi/gomega` assertion library.
- **Execution**: Run `make test-unit`.
- **Arguments**: To pass flags to `go test`, use the `ARGS` variable.
  - Example: `make test-unit ARGS='-run="TestConfig_validateFlags"'`

## Go Module Management
- **Verification and Tidying**: To ensure Go module integrity and cleanliness, the following commands will be run automatically after any Go module updates:
  - `go mod verify`: Verifies the module dependencies.
  - `go mod tidy -v`: Cleans up the `go.mod` and `go.sum` files, removing unused dependencies.
  - `go mod vendor`: Updates the `vendor/` directory to reflect the current module dependencies.
- **Automation**: This sequence is now part of the standard development workflow and will be enforced during module updates.

## Architecture (CLI)
- **Library**: `cobra`.
- **Pattern**: Commands are defined as structs in `pkg/subcmd` (e.g., `Config` struct) implementing a common interface.
- **Validation**: Specific `validateFlags()` methods are used for flag logic (e.g., mutual exclusion) rather than just Cobra's built-in validators.
- **Dependencies**: Services like `k8s.Kube` and `chartfs.ChartFS` are injected via constructors.

## Repository Structure

The repository packages are organized by dependency tier:

### Low-Level Packages
- **`pkg/flags/`**: Global CLI flags infrastructure shared across subcommands.
- **`pkg/k8s/`**: Kubernetes client wrapper and related machinery.
- **`pkg/monitor/`**: Logic for streaming changes on Kubernetes resources managed by the installer.
- **`pkg/engine/`**: Template rendering engine (Helm values with Sprig functions).

### Mid-Level Packages
- **`pkg/config/`**: Configuration management (config.yaml handling).
- **`pkg/deployer/`**: Helm deployment abstraction (install/upgrade/test operations).
- **`pkg/integration/`**: Integration interface and helpers for connecting to external services like GitHub, Quay, etc.
- **`pkg/integrations/`**: Integration lifecycle manager.
- **`pkg/api/`**: Core API types and interfaces (AppContext, SubCommand, etc.).

### High-Level Packages
- **`pkg/resolver/`**: Dependency topology resolution using Helm chart annotations. Defines the sequence of dependencies (Helm Charts) and their relationships.
- **`pkg/hooks/`**: Pre/post deployment hooks execution.
- **`pkg/installer/`**: Main installation orchestrator coordinating the entire deployment lifecycle.

### CLI and MCP Interface Packages
- **`pkg/subcmd/`**: CLI subcommands (config, deploy, template, topology, integration, mcpserver, etc.). Includes all integration subcommands.
- **`pkg/mcptools/`**: MCP server tools exposing installer functionality via MCP protocol.

### Application Bootstrap
- **`pkg/framework/`**: Application runtime that wires together all components. Creates App type, manages lifecycle, and registers subcommands.

### Application-Specific Directories
- **`cmd/tssc/`**: Contains the `main` package - CLI entry point.
- **`installer/`**: Helm charts (dependencies), default installer configuration, and `values.yaml.tpl` file. These resources are embedded in the executable using `go:embed` directive.
- **`docs/`**: Project documentation files.
- **`integration-tests/`**, **`test/`**: Test suites.
- **`scripts/`**, **`hack/`**: Automation and helper scripts.
- **`ci/`**, **`.github/`**, **`.tekton/`**: CI/CD pipeline configurations.
- **`image/`**, **`Dockerfile`**: Container image artifacts.

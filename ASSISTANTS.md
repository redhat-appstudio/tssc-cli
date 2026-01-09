# Project: `github.com/redhat-appstudio/tssc-cli`

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
- **`cmd/tssc/`**: Contains the `main` package.
- **`docs/`**: Contains project documentation files.
- **`installer/`**: Contains the Helm charts (dependencies), default installer configuration and the `values.yaml.tpl` file. These resources are embedded in the executable using `go:embed` directive.
- **`pkg/chartfs`**: Contains a `io.FS` compatible abstraction that provides access to the `installer` directory resources. 
- **`pkg/config/`**: Contains configuration handling logic.
- **`pkg/engine/`**: Contains the Helm engine, responsible for installing the charts, templating functions and managing the lifecycle of the installation. Uses Helm SDK.
- **`pkg/installer`**: Contains the overall installer logic.
- **`pkg/integration`**: Contains the integration definitions. Integrations are used to connect to various external services like GitHub, Quay, etc.
- **`pkg/integrations`**: Contains the logic to manage integrations lifecycle.
- **`pkg/k8s`**: Contains the Kubernetes related machinery.
- **`pkg/mcpserver`**: Contains the MCP (Model Context Protocol) server instance, responsible for the communication with the agentic LLM client, via a API. It also contains the `pkg/mcpserver/instructions.md`, the set of instructions passed to the LLM model to drive the installation process through MCP tools.
- **`pkg/mcptools`**: Contains the MCP tools, which expose the whole installer features through MCP API.
- **`pkg/monitor`**: Contains the logic for streaming the changes on certain Kubernetes resources managed by the installer.
- **`pkg/resolver`**: Contains the logic to define the dependency topology, which is the main concept behind the installer's architecture. Defines the sequence of dependencies (Helm Charts) as well as the relationships between them and required integrations.

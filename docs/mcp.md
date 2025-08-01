`tssc`: Model Context Protocol Server (MCP)
-------------------------------------------

# Usage

Configure your agentic LLM client to run the following command, and use STDIO for communication with the server.

```sh
tssc mcp-server --image="ghcr.io/redhat-appstudio/tssc:latest"
```

# Instructions

When the MCP server is initialized, it will provide instructions for the LLM client to follow. The instructions are documented [here](../pkg/mcpserver/instructions.md).

The instructions provide a clear overview of what the MCP server is designed to do, how it works, and the expected sequence of tools to follow.

# Tools

The following MCP tools are exposed by the MCP server.

## Configuration

This section covers for the features in `tssc config` subcommand.

### `tssc_config_get`

- *Description*: Get the existing TSSC configuration in the cluster, or return the default if none exists yet. Use the default configuration as the reference to create a new TSSC configuration for the cluster.
- *Arguments*: None.

### `tssc_config_create`

- *Description*: Create a new TSSC configuration in the cluster, in case none exists yet. Use the defaults as the reference to create a new TSSC cluster configuration.
- *Arguments*:
    - **namespace** (string):
        - **Description**: The main namespace for TSSC (`.tssc.namespace`), where Red Hat Developer Hub (DH) and other fundamental services will be deployed.
        - **Default**: "tssc".
    - **setting** (object):
        - **Description**: The global settings object for TSSC (`.tssc.settings{}`). When empty the default settings will be used.

## Integration

Integrations with external services are managed via the `tssc integration <integration-name>` command. Since these integrations often require sensitive information, credentials, the MCP server will not handle them directly. Instead, it will provide the user with the exact `tssc integration` commands needed to configure the integration.

### `tssc_integration_list`

- *Description*: List the TSSC integrations available for the user. Certain integrations are required for certain features, make sure to configure the integrations accordingly.
- *Arguments*: None.

## Deploy

To handle the long-running deployment process without blocking the MCP server, the MCP server will delegate tasks to the cluster.

The `tssc` design maintains a single deployment per cluster, by centralising the installer configuration on a unique `ConfigMap`, in order to have a single RHADS installation per cluster.

Following the same principle, the MCP server will generate a Kubernetes Job to runs the `tssc` container image, more specifically the `tssc deploy` subcommand, which proceeds with the pre-defined sequence of Helm chart deployments, the RHADS deployment itself.

### `tssc_deploy_status`

- *Description*: Reports the status of the TSSC deploy Job running in the cluster.
- *Arguments*: None.

### `tssc_deploy`

- *Description*: Deploys TSSC components to the cluster, uses the cluster configuration to deploy the TSSC components sequentially.
- *Arguments*: None.

package mcptools

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/redhat-appstudio/tssc/pkg/config"
	"github.com/redhat-appstudio/tssc/pkg/k8s"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// ConfigTools represents a set of tools for managing the TSSC configuration in a
// Kubernetes cluster via MCP tools. Each tool is a function that handles a
// specific role in the TSSC configuration lifecycle. Analogous to the "tssc
// config" subcommand it uses the ConfigManager to manage the TSSC configuration
// in the cluster.
type ConfigTools struct {
	logger *slog.Logger             // application logger
	cm     *config.ConfigMapManager // cluster config manager
	kube   *k8s.Kube                // kubernetes client

	defaultDependencies config.Dependencies // default config dependencies
	defaultCfg          *config.Config      // default config (embedded)
}

const (
	// NamespaceArg namespace argument.
	NamespaceArg = "namespace"
	// SettingsArg settings argument.
	SettingsArg = "setting"
)

// getHandler similar to "tssc config --get" subcommand it returns a existing TSSC
// cluster configuration. If no such configuration exists it returns the
// installer's default.
func (c *ConfigTools) getHandler(
	ctx context.Context,
	ctr mcp.CallToolRequest,
) (*mcp.CallToolResult, error) {
	cfg, err := c.cm.GetConfig(ctx)
	// The cluster is already configured, showing the user the existing
	// configuration as text.
	if err == nil {
		return mcp.NewToolResultText(
			fmt.Sprintf("Current TSSC configuration:\n%s", cfg.String()),
		), nil
	}

	// Return error when different than configuration not found.
	if !errors.Is(err, config.ErrConfigMapNotFound) {
		return nil, err
	}

	// The cluster is not configured yet, showing the user a default configuration
	// and hints on how to proceed.
	if cfg, err = config.NewConfigDefault(); err != nil {
		return nil, err
	}

	// Using the data structure instead of the original configuration payload to
	// avoid lists of dependencies that might be confusing.
	payload, err := c.defaultCfg.MarshalYAML()
	if err != nil {
		return nil, err
	}

	return mcp.NewToolResultText(
		fmt.Sprintf(`
There's no TSSC configuration in the cluster yet. Carefully consider the default
YAML configuration below as the cluster administrator you need to decide which
products ('.tssc.products[].enabled' attributes) are relevant for the cluster:

---
%s`,
			payload,
		),
	), nil
}

// createHandler handles requests to create a new TSSC configuration, It starts
// from the default config modifying it based on user input.
func (c *ConfigTools) createHandler(
	ctx context.Context,
	ctr mcp.CallToolRequest,
) (*mcp.CallToolResult, error) {
	// Duplicating the default config for the user input changes.
	cfg := *c.defaultCfg

	// Setting the namespace from user input, if provided.
	if ns, ok := ctr.GetArguments()[NamespaceArg].(string); ok {
		cfg.Installer.Namespace = ns
	}

	if settings, ok := ctr.GetArguments()[SettingsArg].(config.Settings); ok {
		cfg.Installer.Settings = settings
	}

	// Making sure the dependencies are back in place.
	cfg.Installer.Dependencies = c.defaultDependencies

	// Ensure the configuration is valid.
	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	// Before creating the cluster configuration, it needs to ensure the OpenShift
	// project exists.
	if err := k8s.EnsureOpenShiftProject(
		ctx,
		c.logger,
		c.kube,
		cfg.Installer.Namespace,
	); err != nil {
		return nil, err
	}

	// Storing the configuration in the cluster.
	if err := c.cm.Create(ctx, &cfg); err != nil {
		return nil, err
	}

	payload, err := cfg.MarshalYAML()
	if err != nil {
		return nil, err
	}
	return mcp.NewToolResultText(
		fmt.Sprintf(`
TSSC has been successfully configured in namespace %s with settings:

---
%s`,
			cfg.Installer.Namespace,
			payload,
		),
	), nil
}

// Init registers the ConfigTools on the provided MCP server instance.
func (c *ConfigTools) Init(s *server.MCPServer) {
	s.AddTools([]server.ServerTool{{
		Tool: mcp.NewTool(
			"tssc_config_get",
			mcp.WithDescription(`
Get the existing TSSC configuration in the cluster, or return the default if none
exists yet. Use the default configuration as the reference to create a new TSSC
configuration for the cluster.`,
			),
		),
		Handler: c.getHandler,
	}, {
		Tool: mcp.NewTool(
			"tssc_config_create",
			mcp.WithDescription(`
Create a new TSSC configuration in the cluster, in case none exists yet. Use the
defaults as the reference to create a new TSSC cluster configuration.`,
			),
			mcp.WithString(
				NamespaceArg,
				mcp.Description(`
The main namespace for TSSC ('.tssc.namespace'), where Red Hat Developer Hub (DH)
and other fundamental services will be deployed.`,
				),
				mcp.DefaultString(c.defaultCfg.Installer.Namespace),
			),
			mcp.WithObject(
				SettingsArg,
				mcp.Description(`
The global settings object for TSSC ('.tssc.settings{}'). When empty the default
settings will be used.
				`),
			),
		),
		Handler: c.createHandler,
	}}...)
}

// NewConfigTools instantiates a new ConfigTools.
func NewConfigTools(
	logger *slog.Logger,
	kube *k8s.Kube,
	cm *config.ConfigMapManager,
) (*ConfigTools, error) {
	// Loading the default configuration to serve as a reference for MCP tools.
	defaultCfg, err := config.NewConfigDefault()
	if err != nil {
		return nil, err
	}

	c := &ConfigTools{
		logger:              logger.With("component", "mcp-config-tools"),
		kube:                kube,
		cm:                  cm,
		defaultDependencies: defaultCfg.Installer.Dependencies,
		defaultCfg:          defaultCfg,
	}
	// Making sure the dependencies are hidden by design.
	c.defaultCfg.Installer.Dependencies = []config.Dependency{}

	return c, nil
}

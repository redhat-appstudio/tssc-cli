package mcptools

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/redhat-appstudio/helmet/api"
	"github.com/redhat-appstudio/helmet/internal/chartfs"
	"github.com/redhat-appstudio/helmet/internal/config"
	"github.com/redhat-appstudio/helmet/internal/k8s"

	"dario.cat/mergo"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// ConfigTools represents a set of tools for managing the configuration in a
// Kubernetes cluster via MCP tools. Each tool is a function that handles a
// specific role in the configuration lifecycle. Analogous to the "config"
// subcommand it uses the ConfigManager to manage the configuration in the
// cluster.
type ConfigTools struct {
	appName string                   // application name for dynamic naming
	logger  *slog.Logger             // application logger
	cfs     *chartfs.ChartFS         // embedded filesystem
	cm      *config.ConfigMapManager // cluster config manager
	kube    *k8s.Kube                // kubernetes client

	defaultCfg *config.Config // default config (embedded)
}

const (
	// configGetSuffix MCP config get tool name suffix.
	configGetSuffix = "_config_get"
	// configInitSuffix initializes the cluster configuration suffix.
	configInitSuffix = "_config_init"
	// configSettingsSuffix manipulates global settings suffix.
	configSettingsSuffix = "_config_settings"
	// configProductEnabledSuffix manipulates the status of a product suffix.
	configProductEnabledSuffix = "_config_product_enabled"
	// configProductNamespaceSuffix manipulates the namespace of a product suffix.
	configProductNamespaceSuffix = "_config_product_namespace"
	// configProductPropertiesSuffix manipulates the properties of a product suffix.
	configProductPropertiesSuffix = "_config_product_properties"
)

// Arguments for the config tools.
const (
	NamespaceArg  = "namespace"
	KeyArg        = "key"
	ValueArg      = "value"
	NameArg       = "name"
	EnabledArg    = "enabled"
	PropertiesArg = "properties"
)

// getHandler similar to "config --get" subcommand it returns a existing
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
			fmt.Sprintf("Current %s configuration:\n%s", c.appName, cfg.String()),
		), nil
	}

	// Return error when different than configuration not found.
	if !errors.Is(err, config.ErrConfigMapNotFound) {
		return nil, err
	}

	// The cluster is not configured yet, showing the user a default configuration
	// and hints on how to proceed.
	if cfg, err = config.NewConfigDefault(c.cfs, ""); err != nil {
		return nil, err
	}

	// Using the data structure instead of the original configuration payload to
	// avoid lists of dependencies that might be confusing.
	payload, err := c.defaultCfg.MarshalYAML()
	if err != nil {
		return nil, err
	}

	return mcp.NewToolResultText(fmt.Sprintf(`
There's no %s configuration in the cluster yet. As the platform engineer,
carefully consider the default YAML configuration below.

---
%s`,
		c.appName,
		payload,
	)), nil
}

// initHandler initializes the cluster configuration using defaults.
func (c *ConfigTools) initHandler(
	ctx context.Context,
	ctr mcp.CallToolRequest,
) (*mcp.CallToolResult, error) {
	// Checking whether the configuration already exists in the cluster.
	if _, err := c.cm.GetConfig(ctx); err == nil {
		return mcp.NewToolResultErrorf(`
The %s configuration already exists in the cluster! Use the %q tool to inspect
the current configuration.`,
			c.appName,
			c.appName+configGetSuffix,
		), nil
	} else if !errors.Is(err, config.ErrConfigMapNotFound) {
		return mcp.NewToolResultErrorFromErr(`
Unable to retrieve the configuration from the cluster!`,
			err,
		), nil
	}

	// Setting the namespace from user input, if provided.
	ns, ok := ctr.GetArguments()[NamespaceArg].(string)
	if !ok || ns == "" {
		return nil, fmt.Errorf("namespace argument is required")
	}

	// Deep-copy the default config to avoid mutating c.defaultCfg.
	payload, err := c.defaultCfg.MarshalYAML()
	if err != nil {
		return nil, err
	}
	cfgPtr, err := config.NewConfigFromBytes(payload, ns)
	if err != nil {
		return nil, err
	}
	cfg := cfgPtr

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
		cfg.Namespace(),
	); err != nil {
		return nil, err
	}

	// Storing the configuration in the cluster.
	if err := c.cm.Create(ctx, cfg); err != nil {
		return nil, err
	}

	return mcp.NewToolResultText(fmt.Sprintf(`
%s default configuration is successfully applied in %q namespace`,
		c.appName,
		cfg.Namespace(),
	)), nil
}

// getConfig Retrieving the existing configuration from the cluster.
func (c *ConfigTools) getConfig(
	ctx context.Context,
) (*config.Config, *mcp.CallToolResult) {
	cfg, err := c.cm.GetConfig(ctx)
	if err != nil {
		return nil, mcp.NewToolResultErrorFromErr(`
Unable to retrieve the configuration from the cluster!`,
			err,
		)
	}
	return cfg, nil
}

// configSettingsHandler handles the configuration settings update request. It
// retrieves the current configuration, merges the provided settings (from the
// 'settings' argument) into the existing installer settings, and updates the
// cluster configuration.
func (c *ConfigTools) configSettingsHandler(
	ctx context.Context,
	ctr mcp.CallToolRequest,
) (*mcp.CallToolResult, error) {
	key, ok := ctr.GetArguments()[KeyArg].(string)
	if !ok || key == "" {
		return mcp.NewToolResultErrorf(`
You must inform the %q argument with the name of the attribute to update!`,
			KeyArg,
		), nil
	}
	value, ok := ctr.GetArguments()[ValueArg].(bool)
	if !ok {
		return mcp.NewToolResultErrorf(`
You must inform the %q argument with the value for the informed key %q!`,
			ValueArg,
			key,
		), nil
	}

	cfg, res := c.getConfig(ctx)
	if res != nil {
		return res, nil
	}

	// Updating the configuration instance and the cluster.
	err := cfg.Set(fmt.Sprintf("tssc.settings.%s", key), value)
	if err != nil {
		return mcp.NewToolResultErrorf(`
Unable to update the existing configuration with informed settings:

    Key: %q
  Value: %v
  Error: %s`,
			key,
			value,
			err,
		), nil
	}
	if err = c.cm.Update(ctx, cfg); err != nil {
		return mcp.NewToolResultErrorFromErr(`
Unable to update the cluster configuration!
`,
			err,
		), nil
	}

	return mcp.NewToolResultText(fmt.Sprintf(`
The cluster is now using the following '.tssc.settings':

%v`,
		cfg.Installer.Settings,
	)), nil
}

// getProduct get the product from the configuration, or returns the MCP error.
func (c *ConfigTools) getProduct(
	cfg *config.Config,
	name string,
) (*config.Product, *mcp.CallToolResult) {
	spec, err := cfg.GetProduct(name)
	if err != nil {
		return nil, mcp.NewToolResultErrorf(`
Unable to find the product name %q: %q`,
			name,
			err,
		)
	}
	return spec, nil
}

// setProduct updates the product configuration by name,using the provided spec
// and persists the changes to the cluster configuration.
func (c *ConfigTools) setProduct(
	ctx context.Context,
	cfg *config.Config,
	name string,
	spec config.Product,
) *mcp.CallToolResult {
	err := cfg.SetProduct(name, spec)
	if err != nil {
		return mcp.NewToolResultErrorf(`
Unable to update product %q: %q`,
			name,
			err,
		)
	}
	if err = c.cm.Update(ctx, cfg); err != nil {
		return mcp.NewToolResultErrorFromErr(`
Unable to update the cluster configuration!
`,
			err,
		)
	}
	return nil
}

// configProductEnableHandler handles requests to enable or disable a specific
// product within the cluster configuration. It requires the 'name' (product name)
// and 'enabled' (boolean status) arguments.
func (c *ConfigTools) configProductEnableHandler(
	ctx context.Context,
	ctr mcp.CallToolRequest,
) (*mcp.CallToolResult, error) {
	name, ok := ctr.GetArguments()[NameArg].(string)
	if !ok {
		return mcp.NewToolResultErrorf(`
You must inform the product name %q argument, in order to toggle its status.`,
			NameArg,
		), nil
	}
	enabled, ok := ctr.GetArguments()[EnabledArg].(bool)
	if !ok {
		return mcp.NewToolResultErrorf(`
You must inform the %q argument, with the status of the product %q`,
			EnabledArg,
			name,
		), nil
	}

	cfg, res := c.getConfig(ctx)
	if res != nil {
		return res, nil
	}

	// Select the product by name.
	spec, res := c.getProduct(cfg, name)
	if res != nil {
		return res, nil
	}
	// Toggle the product status.
	spec.Enabled = enabled

	if res = c.setProduct(ctx, cfg, name, config.Product{Enabled: enabled}); res != nil {
		return res, nil
	}

	return mcp.NewToolResultText(fmt.Sprintf(`
The product %q is toggled to %v, the configuration is applied in the cluster.`,
		name,
		enabled,
	)), nil
}

// configProductNamespaceHandler handles the configuration of a product's
// namespace.  It expects 'name' and 'namespace' arguments.
func (c *ConfigTools) configProductNamespaceHandler(
	ctx context.Context,
	ctr mcp.CallToolRequest,
) (*mcp.CallToolResult, error) {
	name, ok := ctr.GetArguments()[NameArg].(string)
	if !ok {
		return mcp.NewToolResultErrorf(`
You must inform the product name %q argument, in order to change its namespace.`,
			NameArg,
		), nil
	}
	namespace, ok := ctr.GetArguments()[NamespaceArg].(string)
	if !ok {
		return mcp.NewToolResultErrorf(`
You must inform the %q argument for the %q product.`,
			NamespaceArg,
			name,
		), nil
	}

	cfg, res := c.getConfig(ctx)
	if res != nil {
		return res, nil
	}

	// Select the product by name.
	spec, res := c.getProduct(cfg, name)
	if res != nil {
		return res, nil
	}
	// Toggle the namespace on the product spec.
	spec.Namespace = &namespace

	if res = c.setProduct(ctx, cfg, name, *spec); res != nil {
		return res, nil
	}

	return mcp.NewToolResultText(fmt.Sprintf(`
The product %q is using the namespace %q, the configuration is applied in the
cluster.`,
		name,
		namespace,
	)), nil
}

// configProductPropertiesHandler updates the properties of a product
// configuration. It receives the product name and a map of properties to update.
// The properties are merged into the existing configuration, overriding existing
// values if present.
func (c *ConfigTools) configProductPropertiesHandler(
	ctx context.Context,
	ctr mcp.CallToolRequest,
) (*mcp.CallToolResult, error) {
	name, ok := ctr.GetArguments()[NameArg].(string)
	if !ok {
		return mcp.NewToolResultErrorf(`
You must inform the product name %q argument to update its properties.`,
			NameArg,
		), nil
	}
	properties, ok := ctr.GetArguments()[PropertiesArg].(map[string]interface{})
	if !ok {
		return mcp.NewToolResultErrorf(`
You must inform the properties %q argument for the %q product.`,
			PropertiesArg,
			name,
		), nil
	}

	cfg, res := c.getConfig(ctx)
	if res != nil {
		return res, nil
	}

	// Select the product by name.
	spec, res := c.getProduct(cfg, name)
	if res != nil {
		return res, nil
	}
	// Initialize properties when nil.
	if spec.Properties == nil {
		spec.Properties = map[string]interface{}{}
	}

	// Merging current properties with informed.
	err := mergo.Merge(&spec.Properties, properties, mergo.WithOverride)
	if err != nil {
		return mcp.NewToolResultErrorf(`
Unable to merge informed properties with existing configuration!

  Properties: %#v
       Error: %s`,
			properties,
			err,
		), nil
	}

	if res = c.setProduct(ctx, cfg, name, *spec); res != nil {
		return res, nil
	}

	return mcp.NewToolResultText(fmt.Sprintf(`
The product %q has updated properties, and the configuration is applied in the
cluster.

  Properties: %#v`,
		name,
		properties,
	)), nil
}

// Init registers the ConfigTools on the provided MCP server instance.
func (c *ConfigTools) Init(s *server.MCPServer) {
	s.AddTools([]server.ServerTool{{
		Tool: mcp.NewTool(
			c.appName+configGetSuffix,
			mcp.WithDescription(fmt.Sprintf(`
Get the existing %s configuration in the cluster, or return the default if none
exists yet. Use the default configuration as the reference to create a new %s
configuration for the cluster.`,
				c.appName, c.appName,
			)),
		),
		Handler: c.getHandler,
	}, {
		Tool: mcp.NewTool(
			c.appName+configInitSuffix,
			mcp.WithDescription(fmt.Sprintf(`
Initializes the %s default configuration in the informed namespace, in case none
exists yet.`,
				c.appName,
			)),
			mcp.WithString(
				NamespaceArg,
				mcp.Description(fmt.Sprintf(`
The main namespace for %s ('.tssc.namespace'), where Red Hat Developer Hub (DH)
and other fundamental services will be deployed.`,
					c.appName,
				)),
				mcp.DefaultString(c.defaultCfg.Namespace()),
			),
		),
		Handler: c.initHandler,
	}, {
		Tool: mcp.NewTool(
			c.appName+configSettingsSuffix,
			mcp.WithDescription(fmt.Sprintf(`
Modifies the top level settings, '.tssc.settings' in the configuration. It defines
the global settings for the installer applied to all products. Use the tool %q to
inspect the configuration's '.tssc.settings' attributes and their current values,
pay attention to the data type of the values, and make sure they are compatible
with the expected types.`,
				c.appName+configGetSuffix,
			)),
			mcp.WithString(
				KeyArg,
				mcp.Description(`
The key in '.tssc.settings' object to update, for instance "crc".`,
				),
			),
			mcp.WithBoolean(
				ValueArg,
				mcp.Description(`
The value for the informed key in '.tssc.settings' object.`,
				),
			),
		),
		Handler: c.configSettingsHandler,
	}, {
		Tool: mcp.NewTool(
			c.appName+configProductEnabledSuffix,
			mcp.WithDescription(fmt.Sprintf(`
Toggles a product status, enable or disable it a product for the %s installer
scope. The installer will adequate the deployment topology to the enabled
products.`,
				c.appName,
			)),
			mcp.WithString(
				NameArg,
				mcp.Description(`
The product name to update the '.enabled' attribute.`,
				),
			),
			mcp.WithBoolean(
				EnabledArg,
				mcp.Description(`
Boolean value indicating whether the product should be enabled or not.`,
				),
				mcp.DefaultBool(true),
			),
		),
		Handler: c.configProductEnableHandler,
	}, {
		Tool: mcp.NewTool(
			c.appName+configProductNamespaceSuffix,
			mcp.WithDescription(`
Updates the namespace for a given product, which means the primary product
components will take place on the specified Kubernetes namespace.`,
			),
			mcp.WithString(
				NameArg,
				mcp.Description(`
The product name to update the '.namespace' attribute.`,
				),
			),
			mcp.WithString(
				NamespaceArg,
				mcp.Description(`
The namespace where the product components will take place.`,
				),
				mcp.DefaultString(""),
			),
		),
		Handler: c.configProductNamespaceHandler,
	}, {
		Tool: mcp.NewTool(
			c.appName+configProductPropertiesSuffix,
			mcp.WithDescription(`
Updates the properties of a given product, the product '.properties' attributes
will be updated using the informed object.`,
			),
			mcp.WithString(
				NameArg,
				mcp.Description(`
The product name to update its '.properties' attribute.`,
				),
			),
			mcp.WithObject(
				PropertiesArg,
				mcp.Description(`
The properties object with the attributes for the informed product name.`,
				),
			),
		),
		Handler: c.configProductPropertiesHandler,
	}}...)
}

// NewConfigTools instantiates a new ConfigTools.
func NewConfigTools(
	appCtx *api.AppContext,
	logger *slog.Logger,
	cfs *chartfs.ChartFS,
	kube *k8s.Kube,
	cm *config.ConfigMapManager,
) (*ConfigTools, error) {
	// Loading the default configuration to serve as a reference for MCP tools.
	defaultCfg, err := config.NewConfigDefault(cfs, appCtx.Namespace)
	if err != nil {
		return nil, err
	}

	c := &ConfigTools{
		appName:    appCtx.Name,
		logger:     logger.With("component", "mcp-config-tools"),
		cfs:        cfs,
		kube:       kube,
		cm:         cm,
		defaultCfg: defaultCfg,
	}
	return c, nil
}

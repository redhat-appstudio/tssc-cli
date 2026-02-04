package framework

import (
	"github.com/redhat-appstudio/helmet/api"
	"github.com/redhat-appstudio/helmet/internal/mcptools"
)

// Option represents a functional option for the App runtime.
// These options configure runtime dependencies and behavior.
// For application metadata (name, version, etc.), use ContextOption with NewAppContext.
type Option func(*App)

// WithIntegrations sets the supported integrations for the application.
func WithIntegrations(modules ...api.IntegrationModule) Option {
	return func(a *App) {
		a.integrations = append(a.integrations, modules...)
	}
}

// WithMCPImage sets the container image for the MCP server.
func WithMCPImage(image string) Option {
	return func(a *App) {
		a.mcpImage = image
	}
}

// WithMCPToolsBuilder sets the MCP tools builder for the application.
func WithMCPToolsBuilder(builder mcptools.MCPToolsBuilder) Option {
	return func(a *App) {
		a.mcpToolsBuilder = builder
	}
}

// WithInstallerTarball sets the embedded installer tarball for the application.
func WithInstallerTarball(tarball []byte) Option {
	return func(a *App) {
		a.installerTarball = tarball
	}
}

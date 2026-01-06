package framework

import (
	"github.com/redhat-appstudio/tssc-cli/pkg/api"
	"github.com/redhat-appstudio/tssc-cli/pkg/mcptools"
)

// Option represents a functional option for the App.
type Option func(*App)

// WithVersion sets the application version.
func WithVersion(version string) Option {
	return func(a *App) {
		a.Version = version
	}
}

// WithCommitID sets the application commit ID.
func WithCommitID(commitID string) Option {
	return func(a *App) {
		a.CommitID = commitID
	}
}

// WithShortDescription sets the application short description.
func WithShortDescription(short string) Option {
	return func(a *App) {
		a.Short = short
	}
}

// WithLongDescription sets the application long description.
func WithLongDescription(long string) Option {
	return func(a *App) {
		a.Long = long
	}
}

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

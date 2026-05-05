package main

import (
	"fmt"
	"os"

	"github.com/redhat-appstudio/tssc-cli/installer"

	"github.com/redhat-appstudio/helmet/api"
	"github.com/redhat-appstudio/helmet/framework"
)

var (
	// Build-time variables set via ldflags
	version  = "v0.0.0-SNAPSHOT"
	commitID = ""
)

func main() {
	cwd, err := os.Getwd()
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to get the working directory: %v\n", err)
		os.Exit(1)
	}

	// Create TSSC-specific application context (metadata/configuration).
	appCtx := api.NewAppContext(
		"tssc",
		api.WithVersion(version),
		api.WithCommitID(commitID),
		api.WithShortDescription("Trusted Software Supply Chain CLI"),
	)

	// TSSC-specific MCP server image based on build-time commit ID.
	mcpImage := "quay.io/redhat-user-workloads/rhtap-shared-team-tenant/tssc-cli"
	if commitID == "" {
		mcpImage = fmt.Sprintf("%s:latest", mcpImage)
	} else {
		mcpImage = fmt.Sprintf("%s:%s", mcpImage, commitID)
	}

	// Create application runtime from embedded tarball.
	appIntegrations := framework.StandardIntegrations()
	appIntegrations = framework.WithURLProvider(appIntegrations, CustomURLProvider{})
	app, err := framework.NewAppFromTarball(
		appCtx,
		installer.InstallerTarball,
		cwd,
		framework.WithIntegrations(appIntegrations...),
		framework.WithMCPImage(mcpImage),
		framework.WithDistributedInstallerMergeLayout(),
	)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to create application: %v\n", err)
		os.Exit(1)
	}

	printDisclaimer()

	if err := app.Run(); err != nil {
		os.Exit(1)
	}
}
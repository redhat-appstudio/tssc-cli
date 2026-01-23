package main

import (
	"fmt"
	"os"

	"github.com/redhat-appstudio/helmet/pkg/api"
	"github.com/redhat-appstudio/helmet/pkg/chartfs"
	"github.com/redhat-appstudio/helmet/pkg/framework"
	"github.com/redhat-appstudio/helmet/pkg/subcmd"
	"github.com/redhat-appstudio/tssc-cli/installer"
)

var (
	// Build-time variables set via ldflags
	version  = "v0.0.0-SNAPSHOT"
	commitID = ""
)

func main() {
	tfs, err := framework.NewTarFS(installer.InstallerTarball)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to read embedded files: %v\n", err)
		os.Exit(1)
	}
	cwd, err := os.Getwd()
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to get the working directory: %v\n", err)
		os.Exit(1)
	}

	// Create overlay filesystem with embedded tarball (already contains installer
	// directory contents at root) and local filesystem rooted at cwd.
	ofs := chartfs.NewOverlayFS(tfs, os.DirFS(cwd))
	cfs := chartfs.New(ofs)

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

	// Create application runtime with context and dependencies.
	app, err := framework.NewApp(
		appCtx,
		cfs,
		framework.WithIntegrations(subcmd.StandardModules()...),
		framework.WithMCPImage(mcpImage),
		framework.WithInstallerTarball(installer.InstallerTarball),
	)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to create application: %v\n", err)
		os.Exit(1)
	}

	if err := app.Run(); err != nil {
		os.Exit(1)
	}
}

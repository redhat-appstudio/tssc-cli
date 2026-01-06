package main

import (
	"fmt"
	"os"

	"github.com/redhat-appstudio/tssc-cli/installer"
	"github.com/redhat-appstudio/tssc-cli/pkg/chartfs"
	"github.com/redhat-appstudio/tssc-cli/pkg/constants"
	"github.com/redhat-appstudio/tssc-cli/pkg/framework"
	"github.com/redhat-appstudio/tssc-cli/pkg/subcmd"
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

	// Creating a new TSSC application instance using all standard integration
	// modules.
	app, err := framework.NewApp(
		constants.AppName,
		cfs,
		framework.WithShortDescription("Trusted Software Supply Chain CLI"),
		framework.WithIntegrations(subcmd.StandardModules()...),
	)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to create application: %v\n", err)
		os.Exit(1)
	}

	if err := app.Run(); err != nil {
		os.Exit(1)
	}
}

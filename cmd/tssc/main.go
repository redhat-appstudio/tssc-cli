package main

import (
	"fmt"
	"os"

	"github.com/redhat-appstudio/tssc-cli/installer"
	"github.com/redhat-appstudio/tssc-cli/pkg/chartfs"
	"github.com/redhat-appstudio/tssc-cli/pkg/constants"
	"github.com/redhat-appstudio/tssc-cli/pkg/framework"
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
	ofs := &chartfs.OverlayFS{
		Embedded: tfs,
		Local:    os.DirFS(cwd),
	}
	cfs := chartfs.New(ofs)

	// Given the tarball is build based on the "installer" directory, this is the
	// base dir for the new filesystem.
	if bcfs, err := cfs.WithBaseDir("installer"); err == nil {
		cfs = bcfs
	}

	app := framework.NewApp(
		constants.AppName,
		cfs,
		framework.WithShortDescription("Trusted Software Supply Chain CLI"),
	)
	if err := app.Run(); err != nil {
		os.Exit(1)
	}
}

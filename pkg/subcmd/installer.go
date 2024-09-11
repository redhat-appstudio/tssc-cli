package subcmd

import (
	"archive/tar"
	"bytes"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"

	"github.com/redhat-appstudio/rhtap-cli/installer"
	"github.com/redhat-appstudio/rhtap-cli/pkg/flags"

	"github.com/spf13/cobra"
)

// Installer represents the installer subcommand to list and extract the embedded
// resources used for the installation process.
type Installer struct {
	cmd   *cobra.Command // cobra command
	flags *flags.Flags   // global flags

	list    bool   // list the embedded resources
	extract string // extract into a directory
}

var _ Interface = &Installer{}

const installerDesc = `
Shows the embedded installer resources, and extracts them to a directory.

The installer resources can be inspected, and optionally customized for a specific
installation scenario. Later on power up the installation process using the
'deploy' subcommand.

For instance:

	1. Extract the installer resources to a directory:
		$ mkdir /path/to/directory
		$ rhtap-cli installer --list
		$ rhtap-cli installer --extract /path/to/directory

	2. Customize the installer resources on '/path/to/directory' and edit the
		'config.yaml' configuration file (in the same directory).

	3. Deploy the customized installer resources:
		$ rhtap-cli deploy --config /path/to/directory/config.yaml
`

// Cmd exposes the cobra instance.
func (i *Installer) Cmd() *cobra.Command {
	return i.cmd
}

// Complete implements Interface.
func (i *Installer) Complete(_ []string) error {
	return nil
}

// Validate validates the informed flags are correct, and the conditions are met.
func (i *Installer) Validate() error {
	if !i.flags.Embedded {
		return fmt.Errorf("embedded must be enabled")
	}
	if i.list && i.extract != "" {
		return fmt.Errorf("list and extract are mutually exclusive")
	}
	if !i.list && i.extract == "" {
		return fmt.Errorf("either list or extract flags must be set")
	}
	if !i.list && i.extract != "" {
		stat, err := os.Stat(i.extract)
		if err != nil {
			return err
		}
		if !stat.IsDir() {
			return fmt.Errorf("extract target %q must be a directory", i.extract)
		}
	}
	return nil
}

// listResources lists the embedded resources.
func (i *Installer) listResources() error {
	tr := tar.NewReader(bytes.NewReader(installer.InstallerTarball))

	for {
		header, err := tr.Next()
		if err != nil {
			if err == io.EOF {
				break
			}
			return err
		}
		if header.Size == 0 {
			continue
		}
		fmt.Printf("- %q (%d bytes)\n", header.Name, header.Size)
	}
	return nil
}

// extractResources extracts the embedded resources into a the base directory.
func (i *Installer) extractResources() error {
	tr := tar.NewReader(bytes.NewReader(installer.InstallerTarball))

	for {
		header, err := tr.Next()
		if err != nil {
			if err == io.EOF {
				break
			}
			return err
		}

		target := filepath.Join(i.extract, header.Name)

		switch header.Typeflag {
		case tar.TypeDir:
			fmt.Printf("- Creating directory %q\n", target)
			if err := os.MkdirAll(target, os.FileMode(header.Mode)); err != nil {
				return err
			}
		case tar.TypeReg:
			fmt.Printf("- Extracting %q\n", target)
			f, err := os.OpenFile(
				target,
				os.O_CREATE|os.O_WRONLY|os.O_TRUNC,
				os.FileMode(header.Mode),
			)
			if err != nil {
				return err
			}
			defer f.Close()

			if _, err := io.Copy(f, tr); err != nil {
				return err
			}
		case tar.TypeSymlink:
			fmt.Printf("- Creating symlink %q -> %q\n", target, header.Linkname)
			if err := os.Symlink(header.Linkname, target); err != nil {
				return err
			}
		default:
			log.Printf("Unsupported type: %v in %s", header.Typeflag, header.Name)
		}
	}
	return nil
}

// Run lists or extracts the embedded resources.
func (i *Installer) Run() error {
	if i.list {
		return i.listResources()
	}
	return i.extractResources()
}

// NewInstaller creates a new installer subcommand.
func NewInstaller(f *flags.Flags) *Installer {
	i := &Installer{
		cmd: &cobra.Command{
			Use:   "installer",
			Short: "Lists or extract the embedded installer resources",
			Long:  installerDesc,
		},
		flags: f,
	}

	p := i.cmd.PersistentFlags()
	p.BoolVar(&i.list, "list", false, "List the embedded installer resources")
	p.StringVar(
		&i.extract,
		"extract",
		"",
		"Extract the embedded installer resources to a directory",
	)
	return i
}

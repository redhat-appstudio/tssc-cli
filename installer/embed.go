package installer

import _ "embed"

// InstallerTarball is the embedded tarball containing the installer resources,
// like Helm charts, scripts, etc. The file must exist before the build process.
//
//go:embed installer.tar
var InstallerTarball []byte

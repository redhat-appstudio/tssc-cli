package chartfs

import (
	"bytes"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/redhat-appstudio/rhtap-cli/installer"
	"github.com/redhat-appstudio/rhtap-cli/pkg/config"

	"github.com/quay/claircore/pkg/tarfs"
	"helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/chart/loader"
)

// installerDir represents the directory where the installer tarball is stored.
const installerDir = "installer"

// ChartFS represents a file system abstraction which provides the Helm charts
// payload, and as well the "values.yaml.tpl" file.
type ChartFS struct {
	fs      fs.FS  // file system
	baseDir string // base directory path
}

// relativePath returns the relative path for the given file name.
func (c *ChartFS) relativePath(name string) (string, error) {
	// If the file name does not start with the base directory, it means the path
	// is already relative.
	if !strings.HasPrefix(name, c.baseDir) {
		return name, nil
	}

	relPath, err := filepath.Rel(c.baseDir, name)
	if err != nil {
		return "", err
	}
	return relPath, nil
}

// ReadFile reads the file from the file system.
func (c *ChartFS) ReadFile(name string) ([]byte, error) {
	relPath, err := c.relativePath(name)
	if err != nil {
		return nil, err
	}
	return fs.ReadFile(c.fs, relPath)
}

// GetChartForDep returns the Helm chart for the given dependency. It uses
// BufferredFiles to walk through the filesytem and collect the chart files.
func (c *ChartFS) GetChartForDep(dep *config.Dependency) (*chart.Chart, error) {
	bf := NewBufferedFiles(c.fs, dep.Chart)
	if err := fs.WalkDir(c.fs, dep.Chart, bf.Walk); err != nil {
		return nil, err
	}
	return loader.LoadFiles(bf.Files())
}

// NewChartFS instantiates a new ChartFS instance for the given base directory.
func NewChartFS(baseDir string) *ChartFS {
	return &ChartFS{
		fs:      os.DirFS(baseDir),
		baseDir: baseDir,
	}
}

// NewChartFSEmbedded instantiates a new ChartFS instance for the embedded files,
// using a tarball to load the files and thus the base directory is a constant.
func NewChartFSEmbedded() (*ChartFS, error) {
	tfs, err := tarfs.New(bytes.NewReader(installer.InstallerTarball))
	if err != nil {
		return nil, err
	}
	return &ChartFS{
		fs:      tfs,
		baseDir: installerDir,
	}, nil
}

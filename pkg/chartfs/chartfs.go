package chartfs

import (
	"bytes"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/redhat-appstudio/tssc/installer"

	"github.com/quay/claircore/pkg/tarfs"
	"helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/chart/loader"
)

// installerDir represents the directory where the installer tarball is stored.
const embeddedInstallerDir = "installer"

// ChartFS represents a file system abstraction which provides the Helm charts
// payload, and as well the "values.yaml.tpl" file. It uses the embedded tarball
// as data source, and as well, the local file system.
type ChartFS struct {
	embeddedFS      fs.FS  // embedded file system
	embeddedBaseDir string // base directory path
	localFS         fs.FS  // local file system
	localBaseDir    string // base directory path
}

// ErrFailedToReadEmbeddedFiles returned when the tarball is not readable.
var ErrFailedToReadEmbeddedFiles = errors.New("failed to read embedded files")

// relativePath returns the relative path for the given file name.
func (c *ChartFS) relativePath(baseDir, name string) (string, error) {
	// If the file name does not start with the base directory, it means the path
	// is already relative.
	if !strings.HasPrefix(name, baseDir) && name[0] != '/' {
		return name, nil
	}

	relPath, err := filepath.Rel(baseDir, name)
	if err != nil {
		return "", err
	}
	return relPath, nil
}

// readFileFromLocalFS reads the file from the local file system, so using the
// base diretory configured.
func (c *ChartFS) readFileFromLocalFS(name string) ([]byte, error) {
	relPath, err := c.relativePath(c.localBaseDir, name)
	if err != nil {
		return nil, err
	}
	return fs.ReadFile(c.localFS, relPath)
}

// readFileFromEmbeddedFS reads the file from the embedded file system, using the
// known base direcotry for embedded files.
func (c *ChartFS) readFileFromEmbeddedFS(name string) ([]byte, error) {
	relPath, err := c.relativePath(c.embeddedBaseDir, name)
	if err != nil {
		return nil, err
	}
	return fs.ReadFile(c.embeddedFS, relPath)
}

// ReadFile reads the file from the file system.
func (c *ChartFS) ReadFile(name string) ([]byte, error) {
	payload, err := c.readFileFromLocalFS(name)
	if err == nil {
		return payload, nil
	}
	if errors.Is(err, fs.ErrNotExist) {
		return c.readFileFromEmbeddedFS(name)
	}
	return nil, err
}

// walkChartDir walks through the chart directory, and loads the chart files.
func (c *ChartFS) walkChartDir(fsys fs.FS, chartPath string) (*chart.Chart, error) {
	bf := NewBufferedFiles(fsys, chartPath)
	if err := fs.WalkDir(fsys, chartPath, bf.Walk); err != nil {
		return nil, err
	}
	return loader.LoadFiles(bf.Files())
}

// GetChartFiles returns the informed Helm chart path instantiated files.
func (c *ChartFS) GetChartFiles(chartPath string) (*chart.Chart, error) {
	chartFiles, err := c.walkChartDir(c.localFS, chartPath)
	if err == nil {
		return chartFiles, nil
	}
	return c.walkChartDir(c.embeddedFS, chartPath)
}

// NewChartFS instantiates a ChartFS instance using the embedded tarball,
// and the local base directory.
func NewChartFS(baseDir string) (*ChartFS, error) {
	tfs, err := tarfs.New(bytes.NewReader(installer.InstallerTarball))
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrFailedToReadEmbeddedFiles, err)
	}
	return &ChartFS{
		embeddedFS:      tfs,
		embeddedBaseDir: embeddedInstallerDir,
		localFS:         os.DirFS(baseDir),
		localBaseDir:    baseDir,
	}, nil
}

// NewChartFSForCWD instantiates a ChartFS instance using the current working
// directory.
func NewChartFSForCWD() (*ChartFS, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return nil, err
	}
	return NewChartFS(cwd)
}

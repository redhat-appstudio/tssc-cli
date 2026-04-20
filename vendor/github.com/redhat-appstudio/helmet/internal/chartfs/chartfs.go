package chartfs

import (
	"errors"
	"io/fs"
	"os"
	"path"
	"path/filepath"

	"github.com/redhat-appstudio/helmet/internal/constants"

	"helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/chart/loader"
	"helm.sh/helm/v3/pkg/chartutil"
)

// ChartFS represents a file system abstraction which provides the Helm charts
// payload, and as well the "values.yaml.tpl" file. It uses an underlying fs.FS
// as data source.
type ChartFS struct {
	fsys fs.FS // overlay filesystem
}

// ReadFile reads the file from the file system.
// It supports absolute paths (read from the OS filesystem), relative paths
// that exist in the OS filesystem (converted to absolute), and relative paths
// from the embedded filesystem.
func (c *ChartFS) ReadFile(name string) ([]byte, error) {
	// Absolute paths are always read from the OS filesystem
	// For relative paths, try to convert to absolute
	// Check if file exists in OS
	absPath, err := filepath.Abs(name)
	if err == nil {
		if _, statErr := os.Stat(absPath); statErr == nil {
			return os.ReadFile(absPath)
		}
	}
	// Fallback to embedded filesystem
	return fs.ReadFile(c.fsys, name)
}

// LoadedChart pairs a Helm chart with its source directory path relative to the
// ChartFS root (e.g. "charts/product-a").
type LoadedChart struct {
	Path  string
	Chart *chart.Chart
}

// ReadValuesTemplate loads chartDir/values.yaml.tpl when chartDir is non-empty
// and that file exists; otherwise it loads rootValuesPath (typically the
// installer root values.yaml.tpl).
func (c *ChartFS) ReadValuesTemplate(chartDir, rootValuesPath string) ([]byte, string, error) {
	if chartDir != "" {
		p := path.Join(chartDir, constants.ValuesFilename)
		b, err := c.ReadFile(p)
		if err == nil {
			return b, p, nil
		}
		if !errors.Is(err, fs.ErrNotExist) && !errors.Is(err, os.ErrNotExist) {
			return nil, "", err
		}
	}
	b, err := c.ReadFile(rootValuesPath)
	if err != nil {
		return nil, "", err
	}
	return b, rootValuesPath, nil
}

// Open opens the named file. Implements "fs.FS" interface.
func (c *ChartFS) Open(name string) (fs.File, error) {
	return c.fsys.Open(name)
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
	return c.walkChartDir(c.fsys, chartPath)
}

// walkAndFindChartDirs walks through the filesystem and finds all directories
// that contain a Helm chart.
func (c *ChartFS) walkAndFindChartDirs(
	fsys fs.FS,
	root string,
) ([]string, error) {
	chartDirs := []string{}
	fn := func(name string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		// Skipping non-directory entries, we are looking for Helm chart dirs.
		if !d.IsDir() {
			return nil
		}
		// Check if the "Chart.yaml" exists in this directory.
		chartYamlPath := filepath.Join(name, chartutil.ChartfileName)
		if _, err := fs.Stat(fsys, chartYamlPath); err == nil {
			chartDirs = append(chartDirs, name)
		}
		return nil
	}
	if err := fs.WalkDir(fsys, root, fn); err != nil {
		return nil, err
	}
	return chartDirs, nil
}

// GetAllCharts retrieves all Helm charts from the filesystem.
func (c *ChartFS) GetAllCharts() ([]LoadedChart, error) {
	charts := []LoadedChart{}
	chartDirs, err := c.walkAndFindChartDirs(c.fsys, ".")
	if err != nil {
		return nil, err
	}
	for _, chartDir := range chartDirs {
		hc, err := c.GetChartFiles(chartDir)
		if err != nil {
			return nil, err
		}
		charts = append(charts, LoadedChart{
			Path:  filepath.ToSlash(chartDir),
			Chart: hc,
		})
	}
	return charts, nil
}

// WithBaseDir returns a new ChartFS that is rooted at the given base directory.
func (c *ChartFS) WithBaseDir(baseDir string) (*ChartFS, error) {
	sub, err := fs.Sub(c.fsys, baseDir)
	if err != nil {
		return nil, err
	}
	return &ChartFS{fsys: sub}, nil
}

// New creates a ChartFS from any filesystem.
func New(filesystem fs.FS) *ChartFS {
	return &ChartFS{fsys: filesystem}
}

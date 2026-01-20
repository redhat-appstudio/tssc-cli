package chartfs

import (
	"io/fs"
	"path/filepath"

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
func (c *ChartFS) ReadFile(name string) ([]byte, error) {
	return fs.ReadFile(c.fsys, name)
}

// Open opens the named file. Implements "fs.FS" interface.`
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
func (c *ChartFS) GetAllCharts() ([]chart.Chart, error) {
	charts := []chart.Chart{}
	chartDirs, err := c.walkAndFindChartDirs(c.fsys, ".")
	if err != nil {
		return nil, err
	}
	for _, chartDir := range chartDirs {
		chart, err := c.GetChartFiles(chartDir)
		if err != nil {
			return nil, err
		}
		charts = append(charts, *chart)
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

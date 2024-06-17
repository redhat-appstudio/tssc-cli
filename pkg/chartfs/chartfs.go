package chartfs

import (
	"io/fs"
	"os"

	"github.com/redhat-appstudio/rhtap-cli/pkg/config"

	"helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/chart/loader"
)

// ChartFS represents a file system abstraction which provides the Helm charts
// payload, and as well the "values.yaml.tpl" file.
type ChartFS struct {
	fs      fs.FS  // file system
	baseDir string // base directory path
}

// ReadFile reads the file from the file system.
func (c *ChartFS) ReadFile(name string) ([]byte, error) {
	return fs.ReadFile(c.fs, name)
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

// NewChartFSForCWD instantiates a new ChartFS instance for the current working
// directory (".").
func NewChartFSForCWD() *ChartFS {
	return NewChartFS(".")
}

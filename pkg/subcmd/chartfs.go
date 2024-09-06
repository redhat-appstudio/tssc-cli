package subcmd

import (
	"fmt"
	"log/slog"

	"github.com/redhat-appstudio/rhtap-cli/pkg/chartfs"
	"github.com/redhat-appstudio/rhtap-cli/pkg/config"
	"github.com/redhat-appstudio/rhtap-cli/pkg/flags"
)

// newChartFS creates a new ChartFS instance based on the informed configuration
// instance and global flags.
func newChartFS(
	logger *slog.Logger,
	f *flags.Flags,
	cfg *config.Config,
) (*chartfs.ChartFS, error) {
	var cfs *chartfs.ChartFS
	if f.Embedded {
		logger.Debug("Using embedded files...")
		var err error
		if cfs, err = chartfs.NewChartFSEmbedded(); err != nil {
			return nil, fmt.Errorf("failed to read embedded files: %w", err)
		}
	} else {
		baseDir := cfg.GetBaseDir()
		logger.With("base-dir", baseDir).
			Debug("Using configuration file base directory...")
		cfs = chartfs.NewChartFS(baseDir)
	}
	return cfs, nil
}

package runcontext

import (
	"log/slog"

	"github.com/redhat-appstudio/helmet/internal/chartfs"
	"github.com/redhat-appstudio/helmet/internal/k8s"
)

// RunContext carries runtime dependencies for command execution: Kubernetes client,
// chart filesystem, and logger.
type RunContext struct {
	Kube    *k8s.Kube
	ChartFS *chartfs.ChartFS
	Logger  *slog.Logger
}

// NewRunContext builds a RunContext with the given kube, chart filesystem, and logger.
func NewRunContext(kube *k8s.Kube, cfs *chartfs.ChartFS, logger *slog.Logger) *RunContext {
	return &RunContext{
		Kube:    kube,
		ChartFS: cfs,
		Logger:  logger,
	}
}

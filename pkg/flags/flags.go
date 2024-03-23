package flags

import (
	"fmt"
	"log/slog"
	"os"
	"strings"
	"time"

	"github.com/spf13/pflag"
)

// Flags represents the global flags for the application.
type Flags struct {
	Debug          bool          // debug mode
	DryRun         bool          // dry-run mode
	KubeConfigPath string        // path to the kubeconfig file
	LogLevel       *slog.Level   // log verbosity level
	Timeout        time.Duration // helm client timeout
}

// PersistentFlags sets up the global flags.
func (f *Flags) PersistentFlags(p *pflag.FlagSet) {
	p.BoolVar(&f.Debug, "debug", f.Debug, "enable debug mode")
	p.BoolVar(&f.DryRun, "dry-run", f.DryRun, "enable dry-run mode")
	p.StringVar(
		&f.KubeConfigPath,
		"kube-config",
		f.KubeConfigPath,
		"Path to the 'kubeconfig' file",
	)
	p.Var(
		NewLogLevelValue(f.LogLevel),
		"log-level",
		fmt.Sprintf(
			"log verbosity level (default %q)",
			strings.ToLower(f.LogLevel.String()),
		),
	)
	p.Var(
		NewDurationValue(&f.Timeout),
		"timeout",
		fmt.Sprintf(
			"helm client timeout duration (default %q)",
			f.Timeout.String(),
		),
	)
}

// NewFlags instantiates the global flags with default values.
func NewFlags() *Flags {
	defaultLogLevel := slog.LevelWarn
	return &Flags{
		Debug:          false,
		DryRun:         false,
		KubeConfigPath: fmt.Sprintf("%s/.kube/config", os.Getenv("HOME")),
		LogLevel:       &defaultLogLevel,
		Timeout:        15 * time.Minute,
	}
}

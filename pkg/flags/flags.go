package flags

import (
	"fmt"
	"io"
	"log/slog"
	"os/user"
	"path"
	"strings"
	"time"

	"github.com/spf13/pflag"
)

// Flags represents the global flags for the application.
type Flags struct {
	Debug          bool          // debug mode
	DryRun         bool          // dry-run mode
	Embedded       bool          // embedded mode
	KubeConfigPath string        // path to the kubeconfig file
	LogLevel       *slog.Level   // log verbosity level
	Timeout        time.Duration // helm client timeout
}

// PersistentFlags sets up the global flags.
func (f *Flags) PersistentFlags(p *pflag.FlagSet) {
	p.BoolVar(&f.Debug, "debug", f.Debug, "enable debug mode")
	p.BoolVar(&f.DryRun, "dry-run", f.DryRun, "enable dry-run mode")
	p.BoolVar(
		&f.Embedded,
		"embedded",
		f.Embedded,
		"installer resources embedded on the executable",
	)
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

// GetLogger returns a logger instance for flag setting.
func (f *Flags) GetLogger(out io.Writer) *slog.Logger {
	logOpts := &slog.HandlerOptions{Level: f.LogLevel}
	return slog.New(slog.NewTextHandler(out, logOpts))
}

// LoggerWith returns a logger with contextual information.
func (f *Flags) LoggerWith(l *slog.Logger) *slog.Logger {
	return l.With("debug", f.Debug, "dry-run", f.DryRun, "timeout", f.Timeout)
}

// NewFlags instantiates the global flags with default values.
func NewFlags() *Flags {
	// Getting the current user configuration, later on the home directory is used
	// for flag population.
	usr, err := user.Current()
	if err != nil {
		panic(fmt.Errorf("unable to detect current user: %w", err))
	}

	defaultLogLevel := slog.LevelWarn
	return &Flags{
		Debug:          false,
		DryRun:         false,
		Embedded:       true,
		KubeConfigPath: path.Join(usr.HomeDir, ".kube", "config"),
		LogLevel:       &defaultLogLevel,
		Timeout:        15 * time.Minute,
	}
}

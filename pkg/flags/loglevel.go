package flags

import (
	"fmt"
	"log/slog"

	"github.com/spf13/pflag"
)

// LogLevelValue represents slog.Level as a persistent flag.
type LogLevelValue struct {
	logLevel *slog.Level // shared pointer log level
	level    string      // log level name
}

var _ pflag.Value = &LogLevelValue{}

// Set sets the informed level name (string) as a typed "slog.Level" equivalent,
// stored as a valued on the shared pointer.
func (l *LogLevelValue) Set(level string) error {
	switch level {
	case "error":
		*l.logLevel = slog.LevelError
	case "warn":
		*l.logLevel = slog.LevelWarn
	case "info":
		*l.logLevel = slog.LevelInfo
	case "debug":
		*l.logLevel = slog.LevelDebug
	default:
		return fmt.Errorf("unsupported log-level value %q", level)
	}
	// holding the level name for a valid name
	l.level = level
	return nil
}

// String shows the current level name.
func (l *LogLevelValue) String() string {
	return l.level
}

// Type shows the persistent flag type.
func (*LogLevelValue) Type() string {
	return "slog.level"
}

// NewLogLevelValue creates a new instance with the shared slog.Level pointer.
func NewLogLevelValue(logLevel *slog.Level) *LogLevelValue {
	return &LogLevelValue{logLevel: logLevel}
}

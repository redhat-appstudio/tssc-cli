package flags

import (
	"log/slog"
	"testing"
)

func TestLogLevelValue_Set(t *testing.T) {
	tests := []struct {
		name         string
		logLevelName string
		logLevel     slog.Level
		wantErr      bool
	}{{
		name:         "debug level",
		logLevelName: "debug",
		logLevel:     slog.LevelDebug,
		wantErr:      false,
	}, {
		name:         "info level",
		logLevelName: "info",
		logLevel:     slog.LevelInfo,
		wantErr:      false,
	}, {
		name:         "warn level",
		logLevelName: "warn",
		logLevel:     slog.LevelWarn,
		wantErr:      false,
	}, {
		name:         "error level",
		logLevelName: "error",
		logLevel:     slog.LevelError,
		wantErr:      false,
	}, {
		name:         "unknown level",
		logLevelName: "unknown",
		logLevel:     slog.Level(0),
		wantErr:      true,
	}}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var level slog.Level
			l := NewLogLevelValue(&level)

			var err error
			if err = l.Set(tt.logLevelName); (err != nil) != tt.wantErr {
				t.Errorf("LogLevelValue.Set() error = %v, wantErr %v",
					err, tt.wantErr)
			}
			if err != nil {
				return
			}

			if tt.logLevel != level {
				t.Errorf("LogLevelValue.Set() level = %d, expected = %d",
					level, tt.logLevel)
			}
			if tt.logLevelName != l.String() {
				t.Errorf("LogLevelValue.Set() name = %q, expected = %q",
					l.String(), tt.logLevelName)
			}
		})
	}
}

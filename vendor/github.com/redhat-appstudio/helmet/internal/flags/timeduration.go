package flags

import (
	"fmt"
	"time"

	"github.com/spf13/pflag"
)

// DurationValue represents time.Duration as a persistent flag.
type DurationValue struct {
	duration *time.Duration // shared pointer duration
	value    string         // duration value
}

var _ pflag.Value = &DurationValue{}

// Set sets the informed duration value (string) as a typed "time.Duration" equivalent,
// stored as a value on the shared pointer.
func (d *DurationValue) Set(value string) error {
	duration, err := time.ParseDuration(value)
	if err != nil {
		return fmt.Errorf("unsupported duration value %q: %w", value, err)
	}
	*d.duration = duration
	// holding the duration value for a valid value
	d.value = value
	return nil
}

// String shows the current duration value.
func (d *DurationValue) String() string {
	return d.value
}

// Type shows the persistent flag type.
func (*DurationValue) Type() string {
	return "time.Duration"
}

// NewDurationValue creates a new instance with the shared time.Duration pointer.
func NewDurationValue(duration *time.Duration) *DurationValue {
	return &DurationValue{duration: duration}
}

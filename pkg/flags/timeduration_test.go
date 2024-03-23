package flags

import (
	"testing"
	"time"
)

func TestDurationValue_Set(t *testing.T) {
	tests := []struct {
		name     string
		duration string
		wantErr  bool
	}{
		{
			name:     "1s duration",
			duration: "1s",
			wantErr:  false,
		},
		{
			name:     "1m duration",
			duration: "1m",
			wantErr:  false,
		},
		{
			name:     "1h duration",
			duration: "1h",
			wantErr:  false,
		},
		{
			name:     "invalid duration",
			duration: "invalid",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var duration time.Duration
			d := NewDurationValue(&duration)

			err := d.Set(tt.duration)
			if (err != nil) != tt.wantErr {
				t.Errorf("DurationValue.Set() error = %v, wantErr %v", err, tt.wantErr)
			}
			if err != nil {
				return
			}

			expectedDuration, _ := time.ParseDuration(tt.duration)
			if expectedDuration != duration {
				t.Errorf("DurationValue.Set() duration = %v, expected = %v", duration, expectedDuration)
			}
			if tt.duration != d.String() {
				t.Errorf("DurationValue.Set() string = %q, expected = %q", d.String(), tt.duration)
			}
		})
	}
}

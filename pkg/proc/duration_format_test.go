package proc

import (
	"testing"
	"time"
)

func TestFormatDuration(t *testing.T) {
	tests := []struct {
		name     string
		duration time.Duration
		expected string
	}{
		{
			name:     "seconds only",
			duration: 45 * time.Second,
			expected: "45s",
		},
		{
			name:     "minutes and seconds",
			duration: 2*time.Minute + 30*time.Second,
			expected: "2m30s",
		},
		{
			name:     "hours, minutes and seconds",
			duration: 5*time.Hour + 30*time.Minute + 15*time.Second,
			expected: "5h30m15s",
		},
		{
			name:     "days, hours, minutes and seconds",
			duration: 3*24*time.Hour + 12*time.Hour + 45*time.Minute + 30*time.Second,
			expected: "3d12h45m30s",
		},
		{
			name:     "zero duration",
			duration: 0,
			expected: "0s",
		},
		{
			name:     "rounding to seconds",
			duration: 2*time.Minute + 30*time.Second + 500*time.Millisecond,
			expected: "2m31s",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FormatDuration(tt.duration)
			if got != tt.expected {
				t.Errorf("FormatDuration(%v) = %v, want %v", tt.duration, got, tt.expected)
			}
		})
	}
}

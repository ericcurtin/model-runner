package scheduling

import (
	"testing"
	"time"
)

func TestParseIdleTimeout(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		want        time.Duration
		expectError bool
	}{
		{
			name:        "valid 5 minutes",
			input:       "5m",
			want:        5 * time.Minute,
			expectError: false,
		},
		{
			name:        "valid 1 hour",
			input:       "1h",
			want:        1 * time.Hour,
			expectError: false,
		},
		{
			name:        "valid 24 hours",
			input:       "24h",
			want:        24 * time.Hour,
			expectError: false,
		},
		{
			name:        "valid 1 minute (minimum)",
			input:       "1m",
			want:        1 * time.Minute,
			expectError: false,
		},
		{
			name:        "valid 30 minutes",
			input:       "30m",
			want:        30 * time.Minute,
			expectError: false,
		},
		{
			name:        "valid 2 hours",
			input:       "2h",
			want:        2 * time.Hour,
			expectError: false,
		},
		{
			name:        "zero means no timeout (infinite)",
			input:       "0",
			want:        time.Duration(1<<63 - 1), // max duration
			expectError: false,
		},
		{
			name:        "below minimum (30 seconds)",
			input:       "30s",
			want:        0,
			expectError: true,
		},
		{
			name:        "above maximum (25 hours)",
			input:       "25h",
			want:        0,
			expectError: true,
		},
		{
			name:        "invalid format",
			input:       "invalid",
			want:        0,
			expectError: true,
		},
		{
			name:        "empty string",
			input:       "",
			want:        0,
			expectError: true,
		},
		{
			name:        "negative duration",
			input:       "-5m",
			want:        0,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseIdleTimeout(tt.input)
			if (err != nil) != tt.expectError {
				t.Errorf("parseIdleTimeout() error = %v, expectError %v", err, tt.expectError)
				return
			}
			if !tt.expectError && got != tt.want {
				t.Errorf("parseIdleTimeout() = %v, want %v", got, tt.want)
			}
		})
	}
}

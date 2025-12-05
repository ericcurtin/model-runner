package commands

import (
	"testing"
	"time"
)

func TestBenchCmdFlags(t *testing.T) {
	cmd := newBenchCmd()

	// Verify the --prompt flag exists
	promptFlag := cmd.Flags().Lookup("prompt")
	if promptFlag == nil {
		t.Fatal("--prompt flag not found")
	}
	if promptFlag.DefValue != defaultBenchPrompt {
		t.Errorf("Expected default prompt to be %q, got %q", defaultBenchPrompt, promptFlag.DefValue)
	}

	// Verify the --concurrency flag exists with shorthand
	concurrencyFlag := cmd.Flags().Lookup("concurrency")
	if concurrencyFlag == nil {
		t.Fatal("--concurrency flag not found")
	}
	concurrencyFlagShort := cmd.Flags().ShorthandLookup("c")
	if concurrencyFlagShort == nil {
		t.Fatal("-c shorthand flag not found")
	}
	if concurrencyFlag.DefValue != "[1,2,4,8]" {
		t.Errorf("Expected default concurrency to be [1,2,4,8], got %s", concurrencyFlag.DefValue)
	}

	// Verify the --requests flag exists with shorthand
	requestsFlag := cmd.Flags().Lookup("requests")
	if requestsFlag == nil {
		t.Fatal("--requests flag not found")
	}
	requestsFlagShort := cmd.Flags().ShorthandLookup("n")
	if requestsFlagShort == nil {
		t.Fatal("-n shorthand flag not found")
	}
	if requestsFlag.DefValue != "3" {
		t.Errorf("Expected default requests to be 3, got %s", requestsFlag.DefValue)
	}
}

func TestCalculateStats(t *testing.T) {
	tests := []struct {
		name          string
		results       []benchResult
		concurrency   int
		totalDuration time.Duration
		wantSuccess   int
		wantFailed    int
		wantMinDur    time.Duration
		wantMaxDur    time.Duration
	}{
		{
			name: "all successful requests",
			results: []benchResult{
				{Duration: 100 * time.Millisecond, CompletionTokens: 50, TotalTokens: 60},
				{Duration: 200 * time.Millisecond, CompletionTokens: 100, TotalTokens: 110},
				{Duration: 150 * time.Millisecond, CompletionTokens: 75, TotalTokens: 85},
			},
			concurrency:   2,
			totalDuration: 250 * time.Millisecond,
			wantSuccess:   3,
			wantFailed:    0,
			wantMinDur:    100 * time.Millisecond,
			wantMaxDur:    200 * time.Millisecond,
		},
		{
			name: "some failed requests",
			results: []benchResult{
				{Duration: 100 * time.Millisecond, CompletionTokens: 50, TotalTokens: 60},
				{Error: errNotRunning},
				{Duration: 150 * time.Millisecond, CompletionTokens: 75, TotalTokens: 85},
			},
			concurrency:   2,
			totalDuration: 200 * time.Millisecond,
			wantSuccess:   2,
			wantFailed:    1,
			wantMinDur:    100 * time.Millisecond,
			wantMaxDur:    150 * time.Millisecond,
		},
		{
			name: "all failed requests",
			results: []benchResult{
				{Error: errNotRunning},
				{Error: errNotRunning},
			},
			concurrency:   1,
			totalDuration: 100 * time.Millisecond,
			wantSuccess:   0,
			wantFailed:    2,
			wantMinDur:    0,
			wantMaxDur:    0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stats := calculateStats(tt.results, tt.concurrency, tt.totalDuration)

			if stats.SuccessfulReqs != tt.wantSuccess {
				t.Errorf("SuccessfulReqs = %d, want %d", stats.SuccessfulReqs, tt.wantSuccess)
			}
			if stats.FailedReqs != tt.wantFailed {
				t.Errorf("FailedReqs = %d, want %d", stats.FailedReqs, tt.wantFailed)
			}
			if stats.Concurrency != tt.concurrency {
				t.Errorf("Concurrency = %d, want %d", stats.Concurrency, tt.concurrency)
			}
			if stats.TotalRequests != len(tt.results) {
				t.Errorf("TotalRequests = %d, want %d", stats.TotalRequests, len(tt.results))
			}
			if stats.MinDuration != tt.wantMinDur {
				t.Errorf("MinDuration = %v, want %v", stats.MinDuration, tt.wantMinDur)
			}
			if stats.MaxDuration != tt.wantMaxDur {
				t.Errorf("MaxDuration = %v, want %v", stats.MaxDuration, tt.wantMaxDur)
			}
		})
	}
}

func TestFormatDuration(t *testing.T) {
	tests := []struct {
		name     string
		duration time.Duration
		want     string
	}{
		{
			name:     "microseconds",
			duration: 500 * time.Microsecond,
			want:     "500.00µs",
		},
		{
			name:     "milliseconds",
			duration: 50 * time.Millisecond,
			want:     "50.00ms",
		},
		{
			name:     "seconds",
			duration: 2 * time.Second,
			want:     "2.00s",
		},
		{
			name:     "fractional seconds",
			duration: 1500 * time.Millisecond,
			want:     "1.50s",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatDuration(tt.duration)
			if got != tt.want {
				t.Errorf("formatDuration(%v) = %q, want %q", tt.duration, got, tt.want)
			}
		})
	}
}

func TestTruncateString(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		maxLen int
		want   string
	}{
		{
			name:   "shorter than max",
			input:  "hello",
			maxLen: 10,
			want:   "hello",
		},
		{
			name:   "equal to max",
			input:  "hello",
			maxLen: 5,
			want:   "hello",
		},
		{
			name:   "longer than max",
			input:  "hello world",
			maxLen: 8,
			want:   "hello...",
		},
		{
			name:   "empty string",
			input:  "",
			maxLen: 5,
			want:   "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := truncateString(tt.input, tt.maxLen)
			if got != tt.want {
				t.Errorf("truncateString(%q, %d) = %q, want %q", tt.input, tt.maxLen, got, tt.want)
			}
		})
	}
}

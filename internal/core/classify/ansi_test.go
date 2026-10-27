package classify_test

import (
	"testing"

	"github.com/soundcheck/soundcheck/internal/core/classify"
)

func TestStripANSI(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "empty",
			input:    "",
			expected: "",
		},
		{
			name:     "no escape codes",
			input:    "hello world\nerror 404",
			expected: "hello world\nerror 404",
		},
		{
			name:     "colors and styles",
			input:    "\x1b[31;1mError:\x1b[0m \x1b[32mhealthy\x1b[0m",
			expected: "Error: healthy",
		},
		{
			name:     "cursor and clear codes",
			input:    "\x1b[2J\x1b[HLogging in...\x1b[K",
			expected: "Logging in...",
		},
		{
			name:     "osc title escape",
			input:    "\x1b]0;Terminal Title\x07Running command...",
			expected: "Running command...",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := classify.StripANSI(tt.input)
			if got != tt.expected {
				t.Errorf("StripANSI(%q) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}

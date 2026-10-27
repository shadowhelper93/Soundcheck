// Package classify implements pure evaluation and classification of agent CLI output.
package classify

import (
	"regexp"
)

// ansiRegex matches standard ANSI terminal escape sequences.
var ansiRegex = regexp.MustCompile("\x1b(?:\\][^\x07\x1b]*(?:\x07|\x1b\\\\)|\\[[0-?]*[ -/]*[@-~]|[@-Z\\\\-_])")

// StripANSI removes ANSI escape codes from terminal output.
func StripANSI(s string) string {
	if s == "" {
		return ""
	}
	return ansiRegex.ReplaceAllString(s, "")
}

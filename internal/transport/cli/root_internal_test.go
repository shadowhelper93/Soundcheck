package cli

import (
	"bytes"
	"errors"
	"testing"
)

func TestExtractExitCode(t *testing.T) {
	cases := []struct {
		name     string
		err      error
		expected int
	}{
		{name: "nil error", err: nil, expected: ExitOK},
		{name: "exit error code", err: &ExitError{Code: ExitDegraded}, expected: ExitDegraded},
		{name: "usage error command", err: errors.New("unknown command foo"), expected: ExitUsage},
		{name: "usage error flag", err: errors.New("unknown flag: --foo"), expected: ExitUsage},
		{name: "generic error", err: errors.New("database connection refused"), expected: ExitFailed},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if code := extractExitCode(tc.err); code != tc.expected {
				t.Fatalf("expected code %d, got %d", tc.expected, code)
			}
		})
	}
}

func TestPrintErrorDiagnostic(t *testing.T) {
	var buf bytes.Buffer

	printErrorDiagnostic(&buf, nil)
	if buf.Len() != 0 {
		t.Fatalf("expected empty buffer for nil error, got: %s", buf.String())
	}

	buf.Reset()
	printErrorDiagnostic(&buf, &ExitError{Code: ExitFailed, Err: nil})
	if buf.Len() != 0 {
		t.Fatalf("expected empty buffer for ExitError with nil Err, got: %s", buf.String())
	}

	buf.Reset()
	printErrorDiagnostic(&buf, &ExitError{Code: ExitUsage, Err: errors.New("specific error")})
	if buf.String() != "specific error\n" {
		t.Fatalf("expected 'specific error\\n', got: %q", buf.String())
	}

	buf.Reset()
	printErrorDiagnostic(&buf, errors.New("generic error"))
	if buf.String() != "generic error\n" {
		t.Fatalf("expected 'generic error\\n', got: %q", buf.String())
	}
}

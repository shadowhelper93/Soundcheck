package cli

import (
	"bytes"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/creack/pty"
	"github.com/soundcheck/soundcheck/internal/app"
	"github.com/soundcheck/soundcheck/internal/core/domain"
)

type upgradeTransport func(*http.Request) (*http.Response, error)

func (f upgradeTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestUpgradeUsesSoundcheckRepository(t *testing.T) {
	previous := http.DefaultTransport
	defer func() { http.DefaultTransport = previous }()
	var requested string
	http.DefaultTransport = upgradeTransport(func(r *http.Request) (*http.Response, error) {
		requested = r.URL.Path
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(`{"tag_name":"v999.0.0"}`)), Header: make(http.Header)}, nil
	})
	cmd := newUpgradeCmd(&RootOptions{})
	cmd.SetOut(io.Discard)
	cmd.SetArgs([]string{"--check"})
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	if requested != "/repos/shadowhelper93/Soundcheck/releases/latest" {
		t.Fatalf("wrong release path: %s", requested)
	}
}

func TestTerminalPresentation(t *testing.T) {
	tests := []struct {
		name   string
		width  uint16
		render func(io.Writer) error
		logo   bool
	}{
		{"wide doctor", 80, func(w io.Writer) error { return outputDoctor(w, &RootOptions{NoColor: true}, app.DoctorReport{}) }, true},
		{"narrow doctor", 60, func(w io.Writer) error { return outputDoctor(w, &RootOptions{NoColor: true}, app.DoctorReport{}) }, false},
		{"json doctor", 80, func(w io.Writer) error { return outputDoctor(w, &RootOptions{JSON: true}, app.DoctorReport{}) }, false},
		{"ndjson doctor", 80, func(w io.Writer) error { return outputDoctor(w, &RootOptions{NDJSON: true}, app.DoctorReport{}) }, false},
		{"root help", 80, func(w io.Writer) error { c := NewRootCmd(&RootOptions{}, nil); c.SetOut(w); return c.Help() }, true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			master, slave, err := pty.Open()
			if err != nil {
				t.Skipf("pty unavailable: %v", err)
			}
			defer func() { _ = master.Close() }()
			if err := pty.Setsize(slave, &pty.Winsize{Cols: tc.width, Rows: 24}); err != nil {
				t.Fatal(err)
			}
			output := make(chan string, 1)
			go func() {
				buf, _ := io.ReadAll(master)
				output <- string(buf)
			}()
			if err := tc.render(slave); err != nil {
				t.Fatal(err)
			}
			_ = slave.Close()
			out := <-output
			if strings.Contains(out, "_____") != tc.logo {
				t.Fatalf("unexpected logo: %s", out)
			}
			if strings.Contains(string(out), "\x1b[") {
				t.Fatalf("unexpected ANSI: %q", out)
			}
		})
	}
}

func TestAdapterStatusShowsLiveProbe(t *testing.T) {
	ad := domain.Adapter{Installed: true, Probes: map[domain.Tier]domain.ProbeResult{
		domain.TierL1: {Status: "passed"},
		domain.TierL2: {Status: "failed", ErrorClass: domain.ErrorClassRateLimited, Message: "quota exhausted"},
	}}
	got := adapterStatus(ad, true)
	if !strings.Contains(got, "AUTH:✓") || !strings.Contains(got, "LIVE:✖") || !strings.Contains(got, "quota exhausted") {
		t.Fatalf("missing distinct probe results: %s", got)
	}
}

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

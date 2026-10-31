package domain_test

import (
	"strings"
	"testing"

	"github.com/soundcheck/soundcheck/internal/core/domain"
)

func probeResult(status string, errClass domain.ErrorClass, msg string) domain.ProbeResult {
	return domain.ProbeResult{Status: status, ErrorClass: errClass, Message: msg}
}

type healthCase struct {
	name       string
	adapter    domain.Adapter
	wantStatus domain.AdapterHealthStatus
	wantReason string // substring; empty means Reason must be empty
}

func checkHealth(t *testing.T, tc healthCase) {
	t.Helper()
	h := domain.EvalHealth(tc.adapter)
	if h.Status != tc.wantStatus {
		t.Errorf("status: got %q, want %q", h.Status, tc.wantStatus)
	}
	if tc.wantReason == "" && h.Reason != "" {
		t.Errorf("reason: got %q, want empty", h.Reason)
	}
	if tc.wantReason != "" && !strings.Contains(h.Reason, tc.wantReason) {
		t.Errorf("reason: got %q, want substring %q", h.Reason, tc.wantReason)
	}
}

func TestEvalHealth_Unavailable(t *testing.T) {
	cases := []healthCase{
		{
			name:       "not installed",
			adapter:    domain.Adapter{Installed: false},
			wantStatus: domain.AdapterHealthUnavailable,
			wantReason: "binary not found",
		},
		{
			name: "L0 failed explicitly",
			adapter: domain.Adapter{
				Installed: true,
				Probes: map[domain.Tier]domain.ProbeResult{
					domain.TierL0: probeResult("failed", domain.ErrorClassNotInstalled, "not found"),
				},
			},
			wantStatus: domain.AdapterHealthUnavailable,
			wantReason: "L0 check failed",
		},
		{
			name: "L1 unauthenticated",
			adapter: domain.Adapter{
				Installed: true,
				Probes: map[domain.Tier]domain.ProbeResult{
					domain.TierL0: probeResult("passed", domain.ErrorClassOK, ""),
					domain.TierL1: probeResult("failed", domain.ErrorClassUnauthenticated, "credentials invalid"),
				},
			},
			wantStatus: domain.AdapterHealthUnavailable,
			wantReason: "L1 auth failed",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) { checkHealth(t, tc) })
	}
}

func TestEvalHealth_Degraded(t *testing.T) {
	cases := []healthCase{
		{
			name: "L2 timeout",
			adapter: domain.Adapter{
				Installed: true,
				Probes: map[domain.Tier]domain.ProbeResult{
					domain.TierL0: probeResult("passed", domain.ErrorClassOK, ""),
					domain.TierL1: probeResult("passed", domain.ErrorClassOK, ""),
					domain.TierL2: probeResult("failed", domain.ErrorClassTimeout, "Probe execution timed out"),
				},
			},
			wantStatus: domain.AdapterHealthDegraded,
			wantReason: "L2 failed",
		},
		{
			name: "L3 model unavailable",
			adapter: domain.Adapter{
				Installed: true,
				Probes: map[domain.Tier]domain.ProbeResult{
					domain.TierL0: probeResult("passed", domain.ErrorClassOK, ""),
					domain.TierL1: probeResult("passed", domain.ErrorClassOK, ""),
					domain.TierL2: probeResult("passed", domain.ErrorClassOK, ""),
					domain.TierL3: probeResult("failed", domain.ErrorClassModelUnavailable, "model not found"),
				},
			},
			wantStatus: domain.AdapterHealthDegraded,
			wantReason: "L3 failed",
		},
		{
			name: "L2 exit-1 (copilot pattern)",
			adapter: domain.Adapter{
				Installed: true,
				Probes: map[domain.Tier]domain.ProbeResult{
					domain.TierL0: probeResult("passed", domain.ErrorClassOK, ""),
					domain.TierL1: probeResult("passed", domain.ErrorClassOK, ""),
					domain.TierL2: probeResult("failed", domain.ErrorClassCorruptedState, "Command failed with exit code 1"),
				},
			},
			wantStatus: domain.AdapterHealthDegraded,
			wantReason: "L2 failed",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) { checkHealth(t, tc) })
	}
}

func TestEvalHealth_Usable(t *testing.T) {
	cases := []healthCase{
		{
			name: "all passed",
			adapter: domain.Adapter{
				Installed: true,
				Probes: map[domain.Tier]domain.ProbeResult{
					domain.TierL0: probeResult("passed", domain.ErrorClassOK, ""),
					domain.TierL1: probeResult("passed", domain.ErrorClassOK, ""),
					domain.TierL2: probeResult("passed", domain.ErrorClassOK, ""),
					domain.TierL3: probeResult("passed", domain.ErrorClassOK, ""),
				},
			},
			wantStatus: domain.AdapterHealthUsable,
		},
		{
			name: "L0 warning installed + L1 passed",
			adapter: domain.Adapter{
				Installed: true,
				Probes: map[domain.Tier]domain.ProbeResult{
					domain.TierL0: probeResult("warning", domain.ErrorClassOK, "version check skipped"),
					domain.TierL1: probeResult("passed", domain.ErrorClassOK, ""),
					domain.TierL2: probeResult("passed", domain.ErrorClassOK, ""),
				},
			},
			wantStatus: domain.AdapterHealthUsable,
		},
		{
			name: "installed, no probes",
			adapter: domain.Adapter{
				Installed: true,
				Probes:    map[domain.Tier]domain.ProbeResult{},
			},
			wantStatus: domain.AdapterHealthUsable,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) { checkHealth(t, tc) })
	}
}

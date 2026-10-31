package domain

// AdapterHealthStatus classifies an adapter's operational state after probing.
type AdapterHealthStatus string

const (
	// AdapterHealthUsable means L0 and L1 passed; the adapter is ready for work.
	AdapterHealthUsable AdapterHealthStatus = "usable"
	// AdapterHealthDegraded means L0 and L1 passed but L2 or L3 failed.
	// The adapter may still run tasks but smoke tests or model availability checks failed.
	AdapterHealthDegraded AdapterHealthStatus = "degraded"
	// AdapterHealthUnavailable means L0 (not installed) or L1 (auth) failed.
	AdapterHealthUnavailable AdapterHealthStatus = "unavailable"
)

// AdapterHealth summarises the operational health of a probed adapter.
type AdapterHealth struct {
	Status AdapterHealthStatus `json:"status"`
	// Reason is non-empty when Status is degraded or unavailable.
	Reason string `json:"reason,omitempty"`
}

// EvalHealth derives the health of an adapter from its recorded probe results.
// Rules (in precedence order):
//  1. Not installed (L0 failed/warning with not_installed) → unavailable.
//  2. L1 failed (unauthenticated or any fail) → unavailable.
//  3. L2 or L3 failed → degraded with the failing tier and message.
//  4. Otherwise → usable.
func EvalHealth(a Adapter) AdapterHealth {
	if !a.Installed {
		return AdapterHealth{Status: AdapterHealthUnavailable, Reason: "binary not found"}
	}

	l0, hasL0 := a.Probes[TierL0]
	if hasL0 && l0.Status == "failed" {
		return AdapterHealth{Status: AdapterHealthUnavailable, Reason: "L0 check failed: " + l0.Message}
	}

	l1, hasL1 := a.Probes[TierL1]
	if hasL1 && l1.Status == "failed" {
		reason := l1.Message
		if reason == "" {
			reason = "authentication check failed"
		}
		return AdapterHealth{Status: AdapterHealthUnavailable, Reason: "L1 auth failed: " + reason}
	}

	for _, tier := range []Tier{TierL2, TierL3} {
		pr, ok := a.Probes[tier]
		if !ok {
			continue
		}
		if pr.Status != "failed" {
			continue
		}
		reason := pr.Message
		if reason == "" {
			reason = string(pr.ErrorClass)
		}
		return AdapterHealth{Status: AdapterHealthDegraded, Reason: string(tier) + " failed: " + reason}
	}

	return AdapterHealth{Status: AdapterHealthUsable}
}

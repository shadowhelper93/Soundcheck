package resolve

import (
	"fmt"

	"github.com/soundcheck/soundcheck/internal/core/domain"
)

// CandidateCheck reports whether one configured role candidate is usable on this host.
type CandidateCheck struct {
	Role      domain.Role `json:"role"`
	Candidate string      `json:"candidate"`
	Verified  bool        `json:"verified"`
	Reason    string      `json:"reason,omitempty"`
}

// ValidateConfig checks every candidate against probed adapters: known agent, known model, installed, authenticated.
func ValidateConfig(cfg domain.Config, adapters []domain.Adapter) []CandidateCheck {
	byID := indexAdapters(adapters)
	var out []CandidateCheck
	for _, role := range roleOrder(cfg) {
		for _, raw := range cfg.Roles[role] {
			ok, reason := checkCandidate(parseCandidate(raw, 0), byID)
			out = append(out, CandidateCheck{Role: role, Candidate: raw, Verified: ok, Reason: reason})
		}
	}
	return out
}

func checkCandidate(c candidate, byID map[string]domain.Adapter) (bool, string) {
	a, ok := byID[c.agentID]
	if !ok {
		return false, fmt.Sprintf("Unknown agent %q.", c.agentID)
	}
	if _, _, pass := checkPresence(a, c); !pass {
		return false, fmt.Sprintf("%s CLI not installed (binary %q not found in PATH).", a.Name, a.Binary)
	}
	if _, _, pass := checkAuth(a, c); !pass {
		return false, fmt.Sprintf("%s CLI is not logged in.", a.Name)
	}
	if c.modelSpec != "" && c.modelSpec != "*" && resolveModelByID(a.Models, c.modelSpec) == nil {
		return false, fmt.Sprintf("Model %q is not in the %s catalog.", c.modelSpec, a.Name)
	}
	return true, ""
}

func resolveModelByID(models []domain.ModelInfo, id string) *domain.ModelInfo {
	for i := range models {
		if models[i].ID == id {
			return &models[i]
		}
	}
	return nil
}

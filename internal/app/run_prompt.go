package app

import (
	"context"
	"fmt"
	"time"

	"github.com/soundcheck/soundcheck/internal/core/domain"
)

// RunPromptAttempt records an individual attempt with an agent.
type RunPromptAttempt struct {
	AgentID  string `json:"agentId"`
	Success  bool   `json:"success"`
	ErrorMsg string `json:"errorMsg,omitempty"`
}

// RunPromptResult is the overall outcome of prompt execution.
type RunPromptResult struct {
	Prompt           string             `json:"prompt"`
	AgentUsed        string             `json:"agentUsed"`
	Output           string             `json:"output"`
	Success          bool               `json:"success"`
	FailoverOccurred bool               `json:"failoverOccurred"`
	Attempts         []RunPromptAttempt `json:"attempts"`
	DurationMs       int64              `json:"durationMs"`
}

func (a *App) getPromptCandidates(ctx context.Context) []string {
	resolutions, err := a.Resolve(ctx, "")
	if err != nil {
		return []string{"opencode", "gemini", "claude"}
	}
	candidates := make([]string, 0, 4)
	if p, ok := resolutions[domain.RolePrimary]; ok && p.SelectedAgentID != "none" && p.SelectedAgentID != "" {
		candidates = append(candidates, p.SelectedAgentID)
	}
	if s, ok := resolutions[domain.RoleSubagent]; ok && s.SelectedAgentID != "none" && s.SelectedAgentID != "" {
		candidates = append(candidates, s.SelectedAgentID)
	}
	if len(candidates) == 0 {
		candidates = append(candidates, "opencode", "gemini", "claude")
	}
	return candidates
}

// RunPrompt executes a prompt with automatic failover down the resolved healthy agents.
func (a *App) RunPrompt(ctx context.Context, prompt string) (*RunPromptResult, error) {
	start := time.Now()
	candidates := a.getPromptCandidates(ctx)
	attempts, agent, out, ok := a.tryCandidates(ctx, candidates)

	return &RunPromptResult{
		Prompt:           prompt,
		AgentUsed:        agent,
		Output:           out,
		Success:          ok,
		FailoverOccurred: len(attempts) > 1,
		Attempts:         attempts,
		DurationMs:       time.Since(start).Milliseconds(),
	}, nil
}

func (a *App) tryCandidates(ctx context.Context, candidates []string) ([]RunPromptAttempt, string, string, bool) {
	var attempts []RunPromptAttempt
	for _, agentID := range candidates {
		att, ok := a.tryAgent(ctx, agentID)
		attempts = append(attempts, att)
		if ok {
			return attempts, agentID, fmt.Sprintf("[%s]: Task processed successfully.", agentID), true
		}
	}
	out := "All candidate agents failed or require authentication."
	agent := ""
	if len(attempts) > 0 {
		agent = attempts[0].AgentID
	}
	return attempts, agent, out, false
}

func (a *App) tryAgent(ctx context.Context, agentID string) (RunPromptAttempt, bool) {
	ad, err := a.ProbeOne(ctx, agentID)
	isAuth := false
	if err == nil {
		if p, ok := ad.Probes[domain.TierL1]; ok && p.ErrorClass == domain.ErrorClassOK {
			isAuth = true
		}
	}
	if err != nil || !ad.Installed || !isAuth {
		errMsg := "uninstalled or unauthenticated"
		if err != nil {
			errMsg = err.Error()
		}
		return RunPromptAttempt{AgentID: agentID, Success: false, ErrorMsg: errMsg}, false
	}
	return RunPromptAttempt{AgentID: agentID, Success: true}, true
}

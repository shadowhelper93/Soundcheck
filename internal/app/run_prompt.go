package app

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/soundcheck/soundcheck/internal/core/classify"
	"github.com/soundcheck/soundcheck/internal/core/domain"
	"github.com/soundcheck/soundcheck/internal/edge/exec"
	"github.com/soundcheck/soundcheck/internal/edge/platform"
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

func (a *App) getPromptCandidates(ctx context.Context, target string) ([]string, error) {
	if target != "" {
		return a.getTargetCandidates(ctx, target)
	}
	return a.getDefaultCandidates(ctx), nil
}

func (a *App) getTargetCandidates(ctx context.Context, target string) ([]string, error) {
	if _, ok := a.defs[target]; ok {
		return []string{target}, nil
	}
	cfg := a.Config()
	targetRole := domain.Role(target)
	candidateSpecs, isRole := cfg.Roles[targetRole]
	if !isRole {
		switch targetRole {
		case domain.RolePrimary, domain.RoleSubagent, domain.RoleAuxiliary, domain.RoleReviewer:
			isRole = true
		}
	}
	if !isRole {
		return nil, fmt.Errorf("unknown adapter or role: %s", target)
	}
	return a.resolveRoleCandidates(ctx, targetRole, candidateSpecs)
}

func (a *App) resolveRoleCandidates(ctx context.Context, role domain.Role, specs []string) ([]string, error) {
	var candidates []string
	seen := make(map[string]bool)
	resolutions, err := a.Resolve(ctx, string(role))
	if err == nil {
		if res, ok := resolutions[role]; ok && res.SelectedAgentID != "" && res.SelectedAgentID != "none" {
			seen[res.SelectedAgentID] = true
			candidates = append(candidates, res.SelectedAgentID)
		}
	}
	for _, spec := range specs {
		agentID := strings.Split(spec, ":")[0]
		if agentID != "" && agentID != "none" && !seen[agentID] {
			seen[agentID] = true
			candidates = append(candidates, agentID)
		}
	}
	if len(candidates) == 0 {
		return nil, fmt.Errorf("no candidates configured for role: %s", role)
	}
	return candidates, nil
}

func (a *App) getDefaultCandidates(ctx context.Context) []string {
	cfg := a.Config()
	candidates := make([]string, 0, 8)
	seen := make(map[string]bool)
	addCandidate := func(id string) {
		if id != "" && id != "none" && !seen[id] {
			seen[id] = true
			candidates = append(candidates, id)
		}
	}
	resolutions, _ := a.Resolve(ctx, "")
	for _, r := range []domain.Role{domain.RolePrimary, domain.RoleReviewer} {
		if res, ok := resolutions[r]; ok {
			addCandidate(res.SelectedAgentID)
		}
	}
	for _, spec := range cfg.Roles[domain.RolePrimary] {
		addCandidate(strings.Split(spec, ":")[0])
	}
	for _, spec := range cfg.Roles[domain.RoleReviewer] {
		addCandidate(strings.Split(spec, ":")[0])
	}
	if len(candidates) == 0 {
		candidates = append(candidates, "opencode", "gemini", "claude")
	}
	return candidates
}

// RunPrompt executes a prompt with automatic failover down the resolved healthy agents.
func (a *App) RunPrompt(ctx context.Context, prompt string, target ...string) (*RunPromptResult, error) {
	start := time.Now()
	tgt := ""
	if len(target) > 0 {
		tgt = target[0]
	}
	candidates, err := a.getPromptCandidates(ctx, tgt)
	if err != nil {
		return nil, err
	}
	attempts, agent, out, ok := a.tryCandidates(ctx, candidates, prompt)

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

func (a *App) tryCandidates(
	ctx context.Context,
	candidates []string,
	prompt string,
) ([]RunPromptAttempt, string, string, bool) {
	var attempts []RunPromptAttempt
	for _, agentID := range candidates {
		att, out, ok := a.tryAgent(ctx, agentID, prompt)
		attempts = append(attempts, att)
		if ok {
			return attempts, agentID, out, true
		}
	}
	out := "All candidate agents failed or require authentication."
	agent := ""
	if len(attempts) > 0 {
		agent = attempts[0].AgentID
	}
	return attempts, agent, out, false
}

func (a *App) buildAgentCmd(agentID string, prompt string) (exec.Cmd, error) {
	def, ok := a.defs[agentID]
	if !ok {
		return exec.Cmd{}, fmt.Errorf("unknown adapter: %s", agentID)
	}
	cmdStr := def.HeadlessCommand
	if cmdStr == "" {
		cmdStr = def.Binary + " -p"
	}
	bin := def.Binary
	var args []string
	switch {
	case bin != "" && cmdStr == bin:
		args = []string{prompt}
	case bin != "" && strings.HasPrefix(cmdStr, bin+" "):
		rest := strings.TrimSpace(strings.TrimPrefix(cmdStr, bin))
		args = append(strings.Fields(rest), prompt)
	default:
		tokens := strings.Fields(cmdStr)
		if len(tokens) == 0 {
			tokens = []string{def.Binary, "-p"}
		}
		bin = tokens[0]
		args = append(tokens[1:], prompt) //nolint:gocritic // distinct destination slice intended
	}
	return exec.Cmd{
		Binary:  bin,
		Args:    args,
		Timeout: 60 * time.Second,
	}, nil
}

func (a *App) tryAgent(ctx context.Context, agentID string, prompt string) (RunPromptAttempt, string, bool) {
	def, ok := a.defs[agentID]
	if !ok {
		return RunPromptAttempt{AgentID: agentID, Success: false, ErrorMsg: "unknown adapter"}, "", false
	}
	if _, err := platform.LookPath(def.Binary); err != nil {
		return RunPromptAttempt{AgentID: agentID, Success: false, ErrorMsg: "uninstalled"}, "", false
	}
	cmd, err := a.buildAgentCmd(agentID, prompt)
	if err != nil {
		return RunPromptAttempt{AgentID: agentID, Success: false, ErrorMsg: err.Error()}, "", false
	}
	return a.runAndClassify(ctx, agentID, cmd)
}

func (a *App) runAndClassify(ctx context.Context, agentID string, cmd exec.Cmd) (RunPromptAttempt, string, bool) {
	res, runErr := a.runner.Run(ctx, cmd)
	if runErr != nil && res.ExitCode == 0 {
		return RunPromptAttempt{AgentID: agentID, Success: false, ErrorMsg: runErr.Error()}, "", false
	}
	def := a.defs[agentID]
	raw := domain.RawOutput{
		AgentID:  agentID,
		ExitCode: res.ExitCode,
		Stdout:   string(res.Stdout),
		Stderr:   string(res.Stderr),
	}
	cl := classify.Classify(raw, def.Quirks)
	if cl.CaughtLie || cl.ErrorClass != domain.ErrorClassOK || res.ExitCode != 0 {
		errMsg := cl.Reason
		if errMsg == "" {
			errMsg = strings.TrimSpace(string(res.Stderr))
			if errMsg == "" {
				errMsg = fmt.Sprintf("exit code %d", res.ExitCode)
			}
		}
		return RunPromptAttempt{AgentID: agentID, Success: false, ErrorMsg: errMsg}, "", false
	}
	out := strings.TrimSpace(string(res.Stdout))
	return RunPromptAttempt{AgentID: agentID, Success: true}, out, true
}

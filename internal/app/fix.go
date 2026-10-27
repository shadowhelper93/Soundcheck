package app

import (
	"context"
	"fmt"
)

// FixResult represents the remediation steps or executed fix for an adapter.
type FixResult struct {
	AdapterID      string `json:"adapterId"`
	RemediationCmd string `json:"remediationCmd"`
	Description    string `json:"description"`
	Success        bool   `json:"success"`
}

// Fix attempts to diagnose and provide/run the remediation for an adapter.
func (a *App) Fix(ctx context.Context, adapterID string) (*FixResult, error) {
	adapter, err := a.ProbeOne(ctx, adapterID)
	if err != nil {
		return nil, fmt.Errorf("unknown or unprobed adapter %q: %w", adapterID, err)
	}

	remediation := ""
	desc := ""

	switch adapterID {
	case "claude":
		remediation = "claude auth login"
		desc = "Run interactive web OAuth login for Claude Code CLI"
	case "codex":
		remediation = "codex auth login"
		desc = "Authenticate OpenAI Codex CLI via web token"
	case "opencode":
		remediation = "opencode auth login"
		desc = "Authenticate OpenCode CLI credentials"
	case "agy":
		remediation = "export GEMINI_API_KEY=<your-key>"
		desc = "Set Gemini or Vertex AI credentials in environment"
	case "gemini":
		remediation = "export GEMINI_API_KEY=<your-key>"
		desc = "Configure API key in environment or run gcloud auth"
	case "copilot":
		remediation = "gh auth login --web"
		desc = "Authorize GitHub Copilot CLI via GitHub CLI"
	default:
		remediation = fmt.Sprintf("%s --help", adapterID)
		desc = fmt.Sprintf("Inspect %s configuration options", adapterID)
	}

	return &FixResult{
		AdapterID:      adapterID,
		RemediationCmd: remediation,
		Description:    desc,
		Success:        adapter.Installed,
	}, nil
}

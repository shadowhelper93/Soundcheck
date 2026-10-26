package classify

import "github.com/soundcheck/soundcheck/internal/core/domain"

func intPtr(i int) *int {
	return &i
}

func claudeQuirks() []domain.Quirk {
	return []domain.Quirk{
		{
			ID:                "claude-79500",
			AgentID:           "claude",
			Name:              "Claude Code Rate Limit False Exit 0",
			DocIssue:          "Issue #79500 (anthropics/claude-code)",
			Category:          "false_exit_code",
			Description:       "API rate limits return subtype:\"success\" with is_error:true and exit code 0! Naive runners think the turn completed successfully.",
			ReferenceURL:      "https://github.com/anthropics/claude-code/issues/79500",
			DiscoveredDate:    "2026-06-12",
			Severity:          "critical",
			Impact:            "Silent workflow pipeline corruption, empty files marked as completed",
			Resolution:        "Inspect JSON is_error and error.type, ignore shell exit code",
			TriggerExitCode:   intPtr(0),
			TriggerRegex:      "(?i)(rate_limit|quota|5-hour)",
			ErrorClass:        domain.ErrorClassRateLimited,
			RewrittenExitCode: 429,
			CooldownMinutes:   30,
			CaughtLie:         true,
		},
		{
			ID:              "claude-74761",
			AgentID:         "claude",
			Name:            "Claude Code Mid-Task Exit 0",
			DocIssue:        "Issue #74761 (anthropics/claude-code)",
			Category:        "false_exit_code",
			Description:     "Headless claude -p exits 0 while loop is mid-task (last assistant message has stop_reason: tool_use).",
			ReferenceURL:    "https://github.com/anthropics/claude-code/issues/74761",
			DiscoveredDate:  "2026-05-18",
			Severity:        "high",
			Impact:          "Partial file edits left unstaged or uncommitted",
			Resolution:      "Check stream-json end_turn event and diff_nonempty gate",
			TriggerExitCode: intPtr(0),
			TriggerRegex:    "\"stop_reason\":\\s*\"tool_use\"",
			ErrorClass:      domain.ErrorClassCorruptedState,
			CaughtLie:       true,
		},
		{
			ID:             "claude-95822",
			AgentID:        "claude",
			Name:           "Claude Auth Polling Exhausts Refresh Tokens",
			DocIssue:       "Issue #95822 (anthropics/claude-code)",
			Category:       "protocol_mismatch",
			Description:    "claude auth status can start OAuth token refresh and exit without saving, invalidating future session tokens.",
			ReferenceURL:   "https://github.com/anthropics/claude-code/issues/95822",
			DiscoveredDate: "2026-07-01",
			Severity:       "high",
			Impact:         "Invalidated future session tokens on repeated health checks",
			Resolution:     "Enforce TTL caching on auth status; do not poll frequently",
			ErrorClass:     domain.ErrorClassUnauthenticated,
		},
	}
}

func codexQuirks() []domain.Quirk {
	return []domain.Quirk{
		{
			ID:                "codex-doctor-exit0",
			AgentID:           "codex",
			Name:              "Codex Doctor False Exit 0 on Auth Failure",
			DocIssue:          "Issue #54831 (openai/codex)",
			Category:          "silent_auth",
			Description:       "Diagnostics command returns exit 0 even when credentials are dead or expired. Must parse stdout JSON rather than exit code.",
			ReferenceURL:      "https://github.com/openai/codex/issues/54831",
			DiscoveredDate:    "2026-04-11",
			Severity:          "critical",
			Impact:            "False green status in CI runners followed by hard crash",
			Resolution:        "Parse JSON doctor report; reject if status is FAILED",
			TriggerExitCode:   intPtr(0),
			TriggerRegex:      "\"doctor_status\":\\s*\"FAILED\"",
			ErrorClass:        domain.ErrorClassUnauthenticated,
			RewrittenExitCode: 401,
			CaughtLie:         true,
		},
		{
			ID:              "codex-issue-10233",
			AgentID:         "codex",
			Name:            "No Headless Quota Inspection API",
			DocIssue:        "Issue #10233 (openai/codex)",
			Category:        "quota_lie",
			Description:     "OpenAI exposes no non-interactive quota command. Quota state can only be inferred reactively from HTTP 429 errors.",
			ReferenceURL:    "https://github.com/openai/codex/issues/10233",
			DiscoveredDate:  "2026-03-02",
			Severity:        "high",
			Impact:          "Cannot pre-check usage balance before expensive runs",
			Resolution:      "Implement LiteLLM cooldown tracker (allowed_fails + cooldown_time)",
			TriggerRegex:    "(?i)(429 Too Many Requests|rate limit)",
			ErrorClass:      domain.ErrorClassRateLimited,
			CooldownMinutes: 15,
		},
	}
}

func agyGeminiQuirks() []domain.Quirk {
	return []domain.Quirk{
		{
			ID:              "antigravity-no-auth-cmd",
			AgentID:         "agy",
			Name:            "Antigravity CLI Interactive Auth Requirement",
			DocIssue:        "Google Antigravity Architecture (June 2026)",
			Category:        "protocol_mismatch",
			Description:     "Antigravity has no non-interactive auth check command; unauthenticated runs output error requiring interactive login.",
			ReferenceURL:    "https://antigravity.google/docs/cli/headless/",
			DiscoveredDate:  "2026-06-18",
			Severity:        "critical",
			Impact:          "Silent lockups or unhandled process exits",
			Resolution:      "Issue micro headless probe; check for .status == SUCCESS (WAITING is not failure)",
			TriggerExitCode: intPtr(1),
			TriggerRegex:    "(?i)(authentication required|agy auth login)",
			ErrorClass:      domain.ErrorClassUnauthenticated,
			CaughtLie:       false,
		},
		{
			ID:                "gemini-base-url-exit41",
			AgentID:           "gemini",
			Name:              "Gemini CLI Custom Base URL Exit 41",
			DocIssue:          "Legacy Gemini CLI (Issue #41)",
			Category:          "false_exit_code",
			Description:       "Setting GOOGLE_GEMINI_BASE_URL causes exit code 41 FatalAuthenticationError even with valid credentials.",
			ReferenceURL:      "https://google-gemini.github.io/gemini-cli/docs/troubleshooting.html",
			DiscoveredDate:    "2026-05-01",
			Severity:          "high",
			Impact:            "Unusable behind enterprise proxy setups",
			Resolution:        "Normalize exit 41 into unauthenticated / migrate to Antigravity",
			TriggerExitCode:   intPtr(41),
			TriggerRegex:      "(?i)FatalAuthenticationError",
			ErrorClass:        domain.ErrorClassUnauthenticated,
			RewrittenExitCode: 429,
			CooldownMinutes:   15,
			CaughtLie:         true,
		},
	}
}

func miscQuirks() []domain.Quirk {
	return []domain.Quirk{
		{
			ID:             "opencode-v1-v2",
			AgentID:        "opencode",
			Name:           "Auth Command Renamed Between v1 and v2",
			DocIssue:       "OpenCode CLI Migration v2.0",
			Category:       "protocol_mismatch",
			Description:    "opencode providers was renamed to opencode auth list --format json in v2.0.",
			ReferenceURL:   "https://opencode.ai/v2/docs/cli/commands/",
			DiscoveredDate: "2026-02-14",
			Severity:       "medium",
			Impact:         "Subprocess crashes with unknown command on older installs",
			Resolution:     "Probe binary version first; dispatch command based on semver",
			ErrorClass:     domain.ErrorClassCorruptedState,
		},
		{
			ID:              "copilot-tool-denials",
			AgentID:         "copilot",
			Name:            "Silent Permission Denials Yield Empty Diffs",
			DocIssue:        "GitHub Copilot CLI tool execution",
			Category:        "protocol_mismatch",
			Description:     "When tool calls are blocked without explicit --allow-tool flags, Copilot exits 0 with no error message and an empty git diff.",
			ReferenceURL:    "https://github.com/github/copilot-cli",
			DiscoveredDate:  "2026-07-20",
			Severity:        "high",
			Impact:          "Runner assumes task completed, but code remains unchanged",
			Resolution:      "Enforce diff_nonempty deterministic gate",
			TriggerExitCode: intPtr(0),
			ErrorClass:      domain.ErrorClassPermissionDenied,
		},
		{
			ID:             "acp-discovery",
			AgentID:        "aider",
			Name:           "Single Code-Path Capability Discovery",
			DocIssue:       "Agent Client Protocol Standard (ACP)",
			Category:       "protocol_mismatch",
			Description:    "Standardized JSON-RPC over stdio. initialize returns agentCapabilities and authMethods for all 42 ACP registry agents.",
			ReferenceURL:   "https://grokipedia.com/page/Agent_Client_Protocol",
			DiscoveredDate: "2026-01-28",
			Severity:       "medium",
			Impact:         "High compatibility without vendor-specific CLI hacks",
			Resolution:     "Negotiate schema in initialize payload",
			ErrorClass:     domain.ErrorClassOK,
		},
	}
}

// DefaultQuirks returns the pure compiled known quirks catalog.
func DefaultQuirks() []domain.Quirk {
	res := make([]domain.Quirk, 0, 10)
	res = append(res, claudeQuirks()...)
	res = append(res, codexQuirks()...)
	res = append(res, agyGeminiQuirks()...)
	res = append(res, miscQuirks()...)
	return res
}

// Package classify implements pure evaluation and classification of agent CLI output.
package classify

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"github.com/soundcheck/soundcheck/internal/core/domain"
)

// Classify inspects raw execution output and quirks to classify the turn outcome.
func Classify(raw domain.RawOutput, quirks []domain.Quirk) domain.Classification {
	jm := parseJSONMap(raw)

	if c, ok := classifyClaude(raw, jm); ok {
		return enrichWithQuirks(c, quirks)
	}
	if c, ok := classifyCodex(raw, jm); ok {
		return enrichWithQuirks(c, quirks)
	}
	if c, ok := classifyAgy(raw, jm); ok {
		return enrichWithQuirks(c, quirks)
	}
	if c, ok := classifyGemini(raw); ok {
		return enrichWithQuirks(c, quirks)
	}
	if c, ok := matchTableQuirk(raw, quirks); ok {
		return enrichWithQuirks(c, quirks)
	}

	c := classifyGeneric(raw)
	return enrichWithQuirks(c, quirks)
}

func parseJSONMap(raw domain.RawOutput) map[string]any {
	if raw.ParsedJSON != nil {
		if m, ok := raw.ParsedJSON.(map[string]any); ok {
			return m
		}
	}
	trimmed := strings.TrimSpace(raw.Stdout)
	if trimmed == "" || !strings.HasPrefix(trimmed, "{") {
		return nil
	}
	var m map[string]any
	if err := json.Unmarshal([]byte(trimmed), &m); err == nil {
		return m
	}
	return nil
}

func classifyClaude(raw domain.RawOutput, jm map[string]any) (domain.Classification, bool) {
	if raw.AgentID != "claude" {
		return domain.Classification{}, false
	}
	if isErr, ok := jm["is_error"].(bool); ok && isErr {
		return classifyClaudeError(raw, jm)
	}
	if stopReason, _ := jm["stop_reason"].(string); stopReason == "tool_use" && jm["result"] == nil {
		return domain.Classification{
			ErrorClass: domain.ErrorClassCorruptedState,
			Reason:     "Detected Issue #74761: claude -p exited 0 mid-task with stop_reason \"tool_use\" before tool completion.",
			CaughtLie:  raw.ExitCode == 0,
			QuirkID:    "claude-74761",
		}, true
	}
	if strings.Contains(raw.Stdout, "5-hour quota") || strings.Contains(raw.Stdout, "rate_limit") {
		return domain.Classification{
			ErrorClass: domain.ErrorClassRateLimited,
			Reason:     "Detected Claude rate limit from text stream.",
			CaughtLie:  raw.ExitCode == 0,
			Rewritten:  429,
			BackoffMin: 30,
			QuirkID:    "claude-79500",
		}, true
	}
	return classifyClaudeSuccess(raw, jm)
}

func classifyClaudeError(raw domain.RawOutput, jm map[string]any) (domain.Classification, bool) {
	errMap, _ := jm["error"].(map[string]any)
	errType, _ := errMap["type"].(string)
	errMsg, _ := errMap["message"].(string)
	if errMsg == "" {
		if res, ok := jm["result"].(string); ok && res != "" {
			errMsg = res
		}
	}

	lowerMsg := strings.ToLower(errMsg)
	isRate := strings.Contains(errType, "rate_limit") ||
		strings.Contains(lowerMsg, "quota") ||
		strings.Contains(lowerMsg, "5-hour") ||
		strings.Contains(lowerMsg, "rate limit") ||
		strings.Contains(lowerMsg, "rate_limit") ||
		strings.Contains(lowerMsg, "[429]") ||
		strings.Contains(raw.Stdout, "rate_limit")

	if isRate {
		return domain.Classification{
			ErrorClass: domain.ErrorClassRateLimited,
			Reason:     fmt.Sprintf("Detected Issue #79500: Claude returned exit code 0 (\"success\"), but JSON contains rate_limit_error: %q", errMsg),
			CaughtLie:  raw.ExitCode == 0,
			Rewritten:  429,
			BackoffMin: 30,
			QuirkID:    "claude-79500",
		}, true
	}
	errClass := domain.ErrorClassUnauthenticated
	if errMsg != "" && !strings.Contains(lowerMsg, "auth") && !strings.Contains(lowerMsg, "key") && !strings.Contains(lowerMsg, "login") {
		errClass = domain.ErrorClassCorruptedState
	}
	return domain.Classification{
		ErrorClass: errClass,
		Reason:     fmt.Sprintf("Claude returned exit 0 with is_error: true - %s", errMsg),
		CaughtLie:  raw.ExitCode == 0,
	}, true
}

func classifyClaudeSuccess(raw domain.RawOutput, jm map[string]any) (domain.Classification, bool) {
	if isErr, ok := jm["is_error"].(bool); ok && !isErr {
		return domain.Classification{
			ErrorClass: domain.ErrorClassOK,
			Reason:     "Claude turn verified cleanly (is_error: false).",
			CaughtLie:  false,
		}, true
	}
	if raw.ExitCode != 0 {
		return domain.Classification{
			ErrorClass: domain.ErrorClassUnauthenticated,
			Reason:     fmt.Sprintf("Claude exited with code %d", raw.ExitCode),
			CaughtLie:  false,
		}, true
	}
	return domain.Classification{
		ErrorClass: domain.ErrorClassOK,
		Reason:     "Claude verified.",
		CaughtLie:  false,
	}, true
}

func classifyCodex(raw domain.RawOutput, jm map[string]any) (domain.Classification, bool) {
	if raw.AgentID != "codex" {
		return domain.Classification{}, false
	}
	if hasCodexDoctorFailure(jm) {
		return domain.Classification{
			ErrorClass: domain.ErrorClassUnauthenticated,
			Reason:     "Detected Codex Doctor Bug: codex doctor exited 0, but internal JSON reports FAILED auth.",
			CaughtLie:  raw.ExitCode == 0,
			Rewritten:  401,
			QuirkID:    "codex-doctor-exit0",
		}, true
	}
	if strings.Contains(raw.Stdout, "429 Too Many Requests") || strings.Contains(strings.ToLower(raw.Stdout), "rate limit") {
		return domain.Classification{
			ErrorClass: domain.ErrorClassRateLimited,
			Reason:     "OpenAI Codex HTTP 429 rate limit reached (reactive detection).",
			CaughtLie:  raw.ExitCode == 0,
			Rewritten:  429,
			BackoffMin: 15,
			QuirkID:    "codex-issue-10233",
		}, true
	}
	return classifyCodexFallback(raw)
}

func hasCodexDoctorFailure(jm map[string]any) bool {
	if status, _ := jm["doctor_status"].(string); status == "FAILED" {
		return true
	}
	checks, _ := jm["checks"].([]any)
	for _, item := range checks {
		checkMap, ok := item.(map[string]any)
		if ok && checkMap["name"] == "authentication" && checkMap["status"] == "FAILED" {
			return true
		}
	}
	return false
}

func classifyCodexFallback(raw domain.RawOutput) (domain.Classification, bool) {
	if raw.ExitCode != 0 {
		return domain.Classification{
			ErrorClass: domain.ErrorClassUnauthenticated,
			Reason:     fmt.Sprintf("Codex exited with code %d. Check CODEX_API_KEY.", raw.ExitCode),
			CaughtLie:  false,
		}, true
	}
	return domain.Classification{
		ErrorClass: domain.ErrorClassOK,
		Reason:     "Codex verified successfully.",
		CaughtLie:  false,
	}, true
}

func classifyAgy(raw domain.RawOutput, jm map[string]any) (domain.Classification, bool) {
	if raw.AgentID != "agy" {
		return domain.Classification{}, false
	}
	if strings.Contains(raw.Stderr, "authentication required") || strings.Contains(raw.Stderr, "agy auth login") {
		return domain.Classification{
			ErrorClass: domain.ErrorClassUnauthenticated,
			Reason:     "Antigravity CLI requires interactive login (no non-interactive auth command).",
			CaughtLie:  false,
			QuirkID:    "antigravity-no-auth-cmd",
		}, true
	}
	if status, _ := jm["status"].(string); status == "SUCCESS" {
		return domain.Classification{
			ErrorClass: domain.ErrorClassOK,
			Reason:     "Antigravity micro-probe passed with status SUCCESS.",
			CaughtLie:  false,
		}, true
	}
	if status, _ := jm["status"].(string); status == "WAITING" {
		return domain.Classification{
			ErrorClass: domain.ErrorClassOK,
			Reason:     "Antigravity in WAITING status (asynchronous step processing, not failure).",
			CaughtLie:  false,
		}, true
	}
	return classifyAgyFallback(raw)
}

func classifyAgyFallback(raw domain.RawOutput) (domain.Classification, bool) {
	if raw.ExitCode != 0 {
		return domain.Classification{
			ErrorClass: domain.ErrorClassUnauthenticated,
			Reason:     fmt.Sprintf("Antigravity probe exited with code %d", raw.ExitCode),
			CaughtLie:  false,
		}, true
	}
	return domain.Classification{
		ErrorClass: domain.ErrorClassOK,
		Reason:     "Antigravity verified.",
		CaughtLie:  false,
	}, true
}

func classifyGemini(raw domain.RawOutput) (domain.Classification, bool) {
	if raw.AgentID != "gemini" {
		return domain.Classification{}, false
	}
	if raw.ExitCode == 41 || strings.Contains(raw.Stderr, "FatalAuthenticationError") {
		return domain.Classification{
			ErrorClass: domain.ErrorClassUnauthenticated,
			Reason:     "Gemini CLI exit 41 (FatalAuthenticationError). Note: Setting GOOGLE_GEMINI_BASE_URL triggers this bug even with a valid key.",
			CaughtLie:  true,
			Rewritten:  429,
			BackoffMin: 15,
			QuirkID:    "gemini-base-url-exit41",
		}, true
	}
	return domain.Classification{}, false
}

func classifyGeneric(raw domain.RawOutput) domain.Classification {
	if raw.ExitCode == 0 {
		return domain.Classification{
			ErrorClass: domain.ErrorClassOK,
			Reason:     "Normal clean execution.",
			CaughtLie:  false,
		}
	}
	lowerErr := strings.ToLower(raw.Stderr)
	if strings.Contains(lowerErr, "auth") || strings.Contains(lowerErr, "unauthorized") || raw.ExitCode == 401 {
		return domain.Classification{
			ErrorClass: domain.ErrorClassUnauthenticated,
			Reason:     "Authentication failed.",
			CaughtLie:  false,
		}
	}
	if strings.Contains(raw.Stderr, "429") || strings.Contains(lowerErr, "rate limit") {
		return domain.Classification{
			ErrorClass: domain.ErrorClassRateLimited,
			Reason:     "Rate limit / quota exceeded.",
			CaughtLie:  false,
		}
	}
	return domain.Classification{
		ErrorClass: domain.ErrorClassCorruptedState,
		Reason:     fmt.Sprintf("Command exited with non-zero code %d", raw.ExitCode),
		CaughtLie:  false,
	}
}

func enrichWithQuirks(c domain.Classification, quirks []domain.Quirk) domain.Classification {
	if len(quirks) == 0 {
		return c
	}
	for _, q := range quirks {
		if q.ID == c.QuirkID {
			if c.Rewritten == 0 && q.RewrittenExitCode != 0 {
				c.Rewritten = q.RewrittenExitCode
			}
			if c.BackoffMin == 0 && q.CooldownMinutes != 0 {
				c.BackoffMin = q.CooldownMinutes
			}
			return c
		}
	}
	return c
}

func matchTableQuirk(raw domain.RawOutput, quirks []domain.Quirk) (domain.Classification, bool) {
	for _, q := range quirks {
		if q.AgentID != "" && q.AgentID != raw.AgentID {
			continue
		}
		if q.TriggerExitCode != nil && *q.TriggerExitCode != raw.ExitCode {
			continue
		}
		if q.TriggerRegex == "" {
			continue
		}
		re, err := regexp.Compile(q.TriggerRegex)
		if err != nil {
			continue
		}
		if re.MatchString(raw.Stdout) || re.MatchString(raw.Stderr) {
			caughtLie := (raw.ExitCode == 0 && q.ErrorClass != domain.ErrorClassOK) ||
				(raw.ExitCode != 0 && q.ErrorClass == domain.ErrorClassOK)
			return domain.Classification{
				ErrorClass: q.ErrorClass,
				Reason:     q.Description,
				CaughtLie:  caughtLie,
				Rewritten:  q.RewrittenExitCode,
				BackoffMin: q.CooldownMinutes,
				QuirkID:    q.ID,
			}, true
		}
	}
	return domain.Classification{}, false
}

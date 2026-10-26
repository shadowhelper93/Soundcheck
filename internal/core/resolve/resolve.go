// Package resolve implements the pure deterministic agent and model role resolution engine.
package resolve

import (
	"fmt"
	"math"
	"sort"
	"strings"
	"time"

	"github.com/soundcheck/soundcheck/internal/core/domain"
)

const defaultCooldownMinutes = 35

type resolveOptions struct {
	ForceCooldownAgentID string
}

// Option configures optional overrides for role resolution.
type Option func(*resolveOptions)

// ResolveOption is an alias for Option for backward-compatibility.
//
//nolint:revive // Explicitly named for contract compatibility.
type ResolveOption = Option

// WithForceCooldownAgentID forces an agent into cooldown during resolution.
func WithForceCooldownAgentID(agentID string) Option {
	return func(o *resolveOptions) {
		o.ForceCooldownAgentID = agentID
	}
}

type resolveContext struct {
	adapterMap      map[string]domain.Adapter
	assignedVendors map[domain.Role]string
	policy          domain.Policy
	now             time.Time
	options         resolveOptions
}

type candidateSelection struct {
	adapter   domain.Adapter
	modelID   string
	reasoning string
}

// DefaultConfig returns the canonical default Soundcheck configuration.
func DefaultConfig() domain.Config {
	return domain.Config{
		Version: 1,
		Roles: map[domain.Role][]string{
			domain.RolePrimary: {
				"claude:claude-3-7-sonnet",
				"codex:gpt-5-codex",
				"opencode:anthropic/claude-sonnet",
				"agy:gemini-2.5-pro",
			},
			domain.RoleSubagent: {
				"claude:claude-3-7-sonnet",
				"opencode:*",
				"agy:gemini-2.5-flash",
			},
			domain.RoleAuxiliary: {
				"claude:claude-3-5-haiku",
				"agy:gemini-2.5-flash",
				"codex:gpt-5-mini",
			},
			domain.RoleReviewer: {
				"codex:*",
				"claude:*",
			},
		},
		Policy: domain.Policy{
			RequireDistinctVendor: []domain.Role{domain.RolePrimary, domain.RoleReviewer},
			MaxCostUSDPerRun:      5.0,
			Probe: domain.ProbePolicy{
				Deep:       false,
				TTLMinutes: 30,
			},
		},
	}
}

// Roles resolves candidate models and agents for roles deterministically.
func Roles(cfg domain.Config, adapters []domain.Adapter, now time.Time, opts ...Option) (map[domain.Role]domain.DetailedResolution, error) {
	var options resolveOptions
	for _, opt := range opts {
		if opt != nil {
			opt(&options)
		}
	}

	ctx := resolveContext{
		adapterMap:      indexAdapters(adapters),
		assignedVendors: make(map[domain.Role]string),
		policy:          cfg.Policy,
		now:             now,
		options:         options,
	}

	roles := roleOrder(cfg)
	results := make(map[domain.Role]domain.DetailedResolution, len(roles))
	for _, role := range roles {
		results[role] = resolveRole(role, cfg.Roles[role], &ctx)
	}

	return results, nil
}

// ResolveRoles is an alias for Roles for contract compatibility.
//
//nolint:revive // Explicitly named for contract compatibility.
func ResolveRoles(cfg domain.Config, adapters []domain.Adapter, now time.Time, opts ...Option) (map[domain.Role]domain.DetailedResolution, error) {
	return Roles(cfg, adapters, now, opts...)
}

// Resolve is a convenience alias for Roles.
func Resolve(cfg domain.Config, adapters []domain.Adapter, now time.Time, opts ...Option) (map[domain.Role]domain.DetailedResolution, error) {
	return Roles(cfg, adapters, now, opts...)
}

func resolveRole(role domain.Role, candidates []string, ctx *resolveContext) domain.DetailedResolution {
	fallbackChain := make([]string, 0)
	steps := make([]domain.ResolverStep, 0)

	for i, cand := range candidates {
		sel, ok := evalCandidate(cand, i, role, ctx, &steps, &fallbackChain)
		if ok {
			ctx.assignedVendors[role] = sel.adapter.Vendor
			return buildSelectedResolution(role, sel, fallbackChain, steps)
		}
	}

	return buildFallbackResolution(role, fallbackChain, steps)
}

func evalCandidate(cand string, idx int, role domain.Role, ctx *resolveContext, steps *[]domain.ResolverStep, fallbacks *[]string) (candidateSelection, bool) {
	agentID, modelSpec := parseCandidate(cand)
	adapter, ok := ctx.adapterMap[agentID]
	if !ok {
		*steps = append(*steps, missingAdapterStep(cand, agentID, modelSpec))
		return candidateSelection{}, false
	}

	step, fb, pass := runCandidateChecks(adapter, cand, modelSpec, role, ctx)
	if !pass {
		*steps = append(*steps, step)
		*fallbacks = append(*fallbacks, fb)
		return candidateSelection{}, false
	}

	sel, step := selectCandidate(adapter, cand, modelSpec, idx)
	*steps = append(*steps, step)
	return sel, true
}

func runCandidateChecks(adapter domain.Adapter, cand, spec string, role domain.Role, ctx *resolveContext) (domain.ResolverStep, string, bool) {
	if step, fb, pass := checkCooldown(adapter, cand, spec, ctx); !pass {
		return step, fb, false
	}
	if step, fb, pass := checkPresence(adapter, cand, spec); !pass {
		return step, fb, false
	}
	if step, fb, pass := checkAuth(adapter, cand, spec); !pass {
		return step, fb, false
	}
	if step, fb, pass := checkVendorConflict(adapter, cand, spec, role, ctx); !pass {
		return step, fb, false
	}
	return domain.ResolverStep{}, "", true
}

func selectCandidate(adapter domain.Adapter, cand, modelSpec string, idx int) (candidateSelection, domain.ResolverStep) {
	targetModelID := concreteModelID(modelSpec, adapter)
	reasoning := fmt.Sprintf("Selected candidate #%d (%s). Passed L0 presence, L1 auth, and vendor diversity constraints.", idx+1, cand)
	step := domain.ResolverStep{
		Candidate: cand,
		AgentID:   adapter.ID,
		ModelID:   targetModelID,
		Status:    domain.StepStatusSelected,
		Details:   reasoning,
	}
	return candidateSelection{
		adapter:   adapter,
		modelID:   targetModelID,
		reasoning: reasoning,
	}, step
}

func checkPresence(adapter domain.Adapter, cand, modelSpec string) (domain.ResolverStep, string, bool) {
	l0, hasL0 := adapter.Probes[domain.TierL0]
	if adapter.Installed && (!hasL0 || l0.Status != "failed") {
		return domain.ResolverStep{}, "", true
	}
	modelID := modelSpec
	if modelID == "" {
		modelID = "*"
	}
	step := domain.ResolverStep{
		Candidate: cand,
		AgentID:   adapter.ID,
		ModelID:   modelID,
		Status:    domain.StepStatusSkippedUninstalled,
		Details:   "L0 Presence check failed: binary not found in PATH.",
	}
	return step, cand + " (not installed)", false
}

func checkCooldown(adapter domain.Adapter, cand, modelSpec string, ctx *resolveContext) (domain.ResolverStep, string, bool) {
	nowMs := ctx.now.UnixMilli()
	cooldownMs := cooldownTimestampMs(adapter.CooldownUntil)
	inCooldown := (cooldownMs > nowMs) || (ctx.options.ForceCooldownAgentID == adapter.ID)
	if !inCooldown {
		return domain.ResolverStep{}, "", true
	}
	modelID := modelSpec
	if modelID == "" {
		modelID = "*"
	}
	rem := remainingCooldownMinutes(cooldownMs, nowMs)
	step := domain.ResolverStep{
		Candidate: cand,
		AgentID:   adapter.ID,
		ModelID:   modelID,
		Status:    domain.StepStatusSkippedCooldown,
		Details:   fmt.Sprintf("In cooldown (%dm remaining) due to rate limit error.", rem),
	}
	return step, cand + " (cooldown)", false
}

func checkAuth(adapter domain.Adapter, cand, modelSpec string) (domain.ResolverStep, string, bool) {
	l1, hasL1 := adapter.Probes[domain.TierL1]
	if !hasL1 || (l1.Status != "failed" && l1.ErrorClass != domain.ErrorClassUnauthenticated) {
		return domain.ResolverStep{}, "", true
	}
	modelID := modelSpec
	if modelID == "" {
		modelID = "*"
	}
	step := domain.ResolverStep{
		Candidate: cand,
		AgentID:   adapter.ID,
		ModelID:   modelID,
		Status:    domain.StepStatusSkippedUnauth,
		Details:   "L1 Auth check failed: credentials invalid or missing.",
	}
	return step, cand + " (unauthenticated)", false
}

func checkVendorConflict(adapter domain.Adapter, cand, modelSpec string, role domain.Role, ctx *resolveContext) (domain.ResolverStep, string, bool) {
	req := ctx.policy.RequireDistinctVendor
	if len(req) < 2 || role != req[1] {
		return domain.ResolverStep{}, "", true
	}
	roleA := req[0]
	vendorA := ctx.assignedVendors[roleA]
	if vendorA == "" || vendorA != adapter.Vendor {
		return domain.ResolverStep{}, "", true
	}
	modelID := modelSpec
	if modelID == "" {
		modelID = "*"
	}
	step := domain.ResolverStep{
		Candidate: cand,
		AgentID:   adapter.ID,
		ModelID:   modelID,
		Status:    domain.StepStatusSkippedVendorConflict,
		Details:   fmt.Sprintf("Policy require_distinct_vendor violated: %s is already from %s.", roleA, adapter.Vendor),
	}
	return step, fmt.Sprintf("%s (vendor conflict: %s)", cand, adapter.Vendor), false
}

func concreteModelID(modelSpec string, adapter domain.Adapter) string {
	if modelSpec != "" && modelSpec != "*" {
		return modelSpec
	}
	if len(adapter.Models) > 0 {
		return adapter.Models[0].ID
	}
	return "default"
}

func missingAdapterStep(cand, agentID, modelSpec string) domain.ResolverStep {
	modelID := modelSpec
	if modelID == "" {
		modelID = "*"
	}
	return domain.ResolverStep{
		Candidate: cand,
		AgentID:   agentID,
		ModelID:   modelID,
		Status:    domain.StepStatusSkippedUninstalled,
		Details:   fmt.Sprintf("Adapter %q is not recognized or installed.", agentID),
	}
}

func parseCandidate(cand string) (string, string) {
	parts := strings.SplitN(cand, ":", 2)
	if len(parts) == 1 {
		return parts[0], ""
	}
	return parts[0], parts[1]
}

func indexAdapters(adapters []domain.Adapter) map[string]domain.Adapter {
	m := make(map[string]domain.Adapter, len(adapters))
	for _, a := range adapters {
		m[a.ID] = a
	}
	return m
}

func roleOrder(cfg domain.Config) []domain.Role {
	standard := []domain.Role{
		domain.RolePrimary,
		domain.RoleSubagent,
		domain.RoleAuxiliary,
		domain.RoleReviewer,
	}
	if len(cfg.Roles) == 0 {
		return standard
	}
	seen := make(map[domain.Role]bool, len(standard))
	for _, r := range standard {
		seen[r] = true
	}
	var extra []domain.Role
	for r := range cfg.Roles {
		if !seen[r] {
			extra = append(extra, r)
		}
	}
	sort.Slice(extra, func(i, j int) bool { return extra[i] < extra[j] })
	return append(standard, extra...)
}

func cooldownTimestampMs(until *int64) int64 {
	if until == nil {
		return 0
	}
	v := *until
	if v > 0 && v < 1e11 {
		return v * 1000
	}
	return v
}

func remainingCooldownMinutes(cooldownMs, nowMs int64) int {
	if cooldownMs <= nowMs {
		return defaultCooldownMinutes
	}
	rem := int(math.Ceil(float64(cooldownMs-nowMs) / 60000.0))
	if rem < 1 {
		return 1
	}
	return rem
}

func resolveModelObj(models []domain.ModelInfo, targetModelID string) *domain.ModelInfo {
	for i := range models {
		if models[i].ID == targetModelID {
			return &models[i]
		}
	}
	if len(models) > 0 {
		return &models[0]
	}
	return nil
}

func modelDisplayAndCost(m *domain.ModelInfo, targetModelID string) (string, float64) {
	if m == nil {
		return targetModelID, 4.0
	}
	displayName := m.DisplayName
	if displayName == "" {
		displayName = targetModelID
	}
	inCost := m.InputCostPerM
	if inCost == 0 {
		inCost = 1.0
	}
	outCost := m.OutputCostPerM
	if outCost == 0 {
		outCost = 3.0
	}
	return displayName, inCost + outCost
}

func buildSelectedResolution(role domain.Role, sel candidateSelection, fallbacks []string, steps []domain.ResolverStep) domain.DetailedResolution {
	modelObj := resolveModelObj(sel.adapter.Models, sel.modelID)
	displayName, cost := modelDisplayAndCost(modelObj, sel.modelID)
	return domain.DetailedResolution{
		Role:              role,
		SelectedAgentID:   sel.adapter.ID,
		SelectedModel:     displayName,
		FallbackChain:     fallbacks,
		Reasoning:         sel.reasoning,
		Vendor:            sel.adapter.Vendor,
		EstimatedCostPerM: cost,
		Healthy:           true,
		Steps:             steps,
	}
}

func buildFallbackResolution(role domain.Role, fallbacks []string, steps []domain.ResolverStep) domain.DetailedResolution {
	return domain.DetailedResolution{
		Role:              role,
		SelectedAgentID:   "none",
		SelectedModel:     "None available",
		FallbackChain:     fallbacks,
		Reasoning:         "All configured candidates exhausted or in cooldown. Manual intervention required.",
		Vendor:            "None",
		EstimatedCostPerM: 0,
		Healthy:           false,
		Steps:             steps,
	}
}

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

type candidate struct {
	raw       string
	agentID   string
	modelSpec string
	index     int
}

type candidateSelection struct {
	adapter   domain.Adapter
	modelID   string
	reasoning string
}

type evalAccumulator struct {
	steps     []domain.ResolverStep
	fallbacks []string
}

func (acc *evalAccumulator) recordSkip(step domain.ResolverStep, fallback string) {
	acc.steps = append(acc.steps, step)
	if fallback != "" {
		acc.fallbacks = append(acc.fallbacks, fallback)
	}
}

func (acc *evalAccumulator) recordSelect(step domain.ResolverStep) {
	acc.steps = append(acc.steps, step)
}

// DefaultConfig returns the canonical default Soundcheck configuration.
func DefaultConfig() domain.Config {
	return domain.Config{
		Version: 1,
		Roles: map[domain.Role][]string{
			domain.RolePrimary: {
				"claude:claude-sonnet-5-5",
				"codex:gpt-4o",
				"opencode:anthropic/claude-sonnet-5-5",
				"agy:gemini-2.5-pro",
			},
			domain.RoleSubagent: {
				"claude:claude-sonnet-5-5",
				"opencode:*",
				"agy:gemini-2.5-flash",
			},
			domain.RoleAuxiliary: {
				"claude:claude-haiku-5-5",
				"agy:gemini-2.5-flash",
				"codex:gpt-4o-mini",
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
	acc := evalAccumulator{
		steps:     make([]domain.ResolverStep, 0),
		fallbacks: make([]string, 0),
	}

	for i, raw := range candidates {
		cand := parseCandidate(raw, i)
		sel, ok := evalCandidate(cand, role, ctx, &acc)
		if ok {
			ctx.assignedVendors[role] = sel.adapter.Vendor
			return buildSelectedResolution(role, sel, acc.fallbacks, acc.steps)
		}
	}

	return buildFallbackResolution(role, acc.fallbacks, acc.steps)
}

func evalCandidate(cand candidate, role domain.Role, ctx *resolveContext, acc *evalAccumulator) (candidateSelection, bool) {
	adapter, ok := ctx.adapterMap[cand.agentID]
	if !ok {
		acc.recordSkip(missingAdapterStep(cand), "")
		return candidateSelection{}, false
	}

	step, fb, pass := runCandidateChecks(adapter, cand, role, ctx)
	if !pass {
		acc.recordSkip(step, fb)
		return candidateSelection{}, false
	}

	sel, step := selectCandidate(adapter, cand)
	acc.recordSelect(step)
	return sel, true
}

func runCandidateChecks(adapter domain.Adapter, cand candidate, role domain.Role, ctx *resolveContext) (domain.ResolverStep, string, bool) {
	if step, fb, pass := checkCooldown(adapter, cand, ctx); !pass {
		return step, fb, false
	}
	if step, fb, pass := checkPresence(adapter, cand); !pass {
		return step, fb, false
	}
	if step, fb, pass := checkAuth(adapter, cand); !pass {
		return step, fb, false
	}
	if step, fb, pass := checkVendorConflict(adapter, cand, role, ctx); !pass {
		return step, fb, false
	}
	return domain.ResolverStep{}, "", true
}

func selectCandidate(adapter domain.Adapter, cand candidate) (candidateSelection, domain.ResolverStep) {
	targetModelID := concreteModelID(cand.modelSpec, adapter)
	reasoning := fmt.Sprintf("Selected candidate #%d (%s). Passed L0 presence, L1 auth, and vendor diversity constraints.", cand.index+1, cand.raw)
	step := domain.ResolverStep{
		Candidate: cand.raw,
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

func checkPresence(adapter domain.Adapter, cand candidate) (domain.ResolverStep, string, bool) {
	l0, hasL0 := adapter.Probes[domain.TierL0]
	if adapter.Installed && (!hasL0 || l0.Status != "failed") {
		return domain.ResolverStep{}, "", true
	}
	step := domain.ResolverStep{
		Candidate: cand.raw,
		AgentID:   adapter.ID,
		ModelID:   candidateModelID(cand.modelSpec),
		Status:    domain.StepStatusSkippedUninstalled,
		Details:   "L0 Presence check failed: binary not found in PATH.",
	}
	return step, cand.raw + " (not installed)", false
}

func checkCooldown(adapter domain.Adapter, cand candidate, ctx *resolveContext) (domain.ResolverStep, string, bool) {
	nowMs := ctx.now.UnixMilli()
	cooldownMs := cooldownTimestampMs(adapter.CooldownUntil)
	inCooldown := (cooldownMs > nowMs) || (ctx.options.ForceCooldownAgentID == adapter.ID)
	if !inCooldown {
		return domain.ResolverStep{}, "", true
	}
	rem := remainingCooldownMinutes(cooldownMs, nowMs)
	step := domain.ResolverStep{
		Candidate: cand.raw,
		AgentID:   adapter.ID,
		ModelID:   candidateModelID(cand.modelSpec),
		Status:    domain.StepStatusSkippedCooldown,
		Details:   fmt.Sprintf("In cooldown (%dm remaining) due to rate limit error.", rem),
	}
	return step, cand.raw + " (cooldown)", false
}

func checkAuth(adapter domain.Adapter, cand candidate) (domain.ResolverStep, string, bool) {
	l1, hasL1 := adapter.Probes[domain.TierL1]
	if !hasL1 || (l1.Status != "failed" && l1.ErrorClass != domain.ErrorClassUnauthenticated) {
		return domain.ResolverStep{}, "", true
	}
	step := domain.ResolverStep{
		Candidate: cand.raw,
		AgentID:   adapter.ID,
		ModelID:   candidateModelID(cand.modelSpec),
		Status:    domain.StepStatusSkippedUnauth,
		Details:   "L1 Auth check failed: credentials invalid or missing.",
	}
	return step, cand.raw + " (unauthenticated)", false
}

func checkVendorConflict(adapter domain.Adapter, cand candidate, role domain.Role, ctx *resolveContext) (domain.ResolverStep, string, bool) {
	req := ctx.policy.RequireDistinctVendor
	if len(req) < 2 || role != req[1] {
		return domain.ResolverStep{}, "", true
	}
	roleA := req[0]
	vendorA := ctx.assignedVendors[roleA]
	if vendorA == "" || vendorA != adapter.Vendor {
		return domain.ResolverStep{}, "", true
	}
	step := domain.ResolverStep{
		Candidate: cand.raw,
		AgentID:   adapter.ID,
		ModelID:   candidateModelID(cand.modelSpec),
		Status:    domain.StepStatusSkippedVendorConflict,
		Details:   fmt.Sprintf("Policy require_distinct_vendor violated: %s is already from %s.", roleA, adapter.Vendor),
	}
	return step, fmt.Sprintf("%s (vendor conflict: %s)", cand.raw, adapter.Vendor), false
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

func candidateModelID(modelSpec string) string {
	if modelSpec == "" {
		return "*"
	}
	return modelSpec
}

func missingAdapterStep(cand candidate) domain.ResolverStep {
	return domain.ResolverStep{
		Candidate: cand.raw,
		AgentID:   cand.agentID,
		ModelID:   candidateModelID(cand.modelSpec),
		Status:    domain.StepStatusSkippedUninstalled,
		Details:   fmt.Sprintf("Adapter %q is not recognized or installed.", cand.agentID),
	}
}

func parseCandidate(cand string, idx int) candidate {
	parts := strings.SplitN(cand, ":", 2)
	spec := ""
	if len(parts) > 1 {
		spec = parts[1]
	}
	return candidate{
		raw:       cand,
		agentID:   parts[0],
		modelSpec: spec,
		index:     idx,
	}
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

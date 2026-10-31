// Package manifest implements pure capability manifest generation for Soundcheck.
package manifest

import (
	"time"

	"github.com/soundcheck/soundcheck/internal/core/domain"
)

const (
	manifestSchemaVersion = "2026.1"
	isoTimestampFormat    = "2006-01-02T15:04:05.000Z"
	defaultNoneValue      = "none"
	defaultUnconfigured   = "No role assignment configured"
)

// BuildManifest constructs a deterministic capability manifest from adapters, resolved roles, and host info.
func BuildManifest(
	adapters []domain.Adapter,
	resolvedRoles map[domain.Role]domain.RoleResolution,
	host domain.HostInfo,
	now time.Time,
) domain.Manifest {
	nowMs := now.UnixMilli()
	agents := make(map[string]domain.ManifestAgent, len(adapters))
	functionalCount, cooldownCount, quirksCount := 0, 0, 0

	for _, a := range adapters {
		agent, isFunc, isCool := buildAgentSummary(a, nowMs)
		agents[a.ID] = agent
		if isFunc {
			functionalCount++
		}
		if isCool {
			cooldownCount++
		}
		quirksCount += len(a.Quirks)
	}

	return domain.Manifest{
		SchemaVersion: manifestSchemaVersion,
		GeneratedAt:   formatGeneratedAt(now),
		Host:          host,
		Summary: domain.ManifestSummary{
			TotalAgents:      len(adapters),
			FunctionalAgents: functionalCount,
			CooldownsActive:  cooldownCount,
			QuirksProtected:  quirksCount,
		},
		Agents:        agents,
		ResolvedRoles: buildRolesSummary(resolvedRoles),
	}
}

// GenerateCapabilityManifest is an alias for BuildManifest for compatibility.
func GenerateCapabilityManifest(
	adapters []domain.Adapter,
	resolvedRoles map[domain.Role]domain.RoleResolution,
	host domain.HostInfo,
	now time.Time,
) domain.Manifest {
	return BuildManifest(adapters, resolvedRoles, host, now)
}

func buildAgentSummary(a domain.Adapter, nowMs int64) (domain.ManifestAgent, bool, bool) {
	isCooldown := a.CooldownUntil != nil && *a.CooldownUntil > nowMs
	l1, hasL1 := a.Probes[domain.TierL1]
	authValid := hasL1 && l1.Status == "passed"
	h := domain.EvalHealth(a)
	isFunctional := h.Status == domain.AdapterHealthUsable && !isCooldown

	errClass := domain.ErrorClassOK
	if l2, hasL2 := a.Probes[domain.TierL2]; hasL2 && l2.ErrorClass != "" {
		errClass = l2.ErrorClass
	}

	models := make([]string, len(a.Models))
	for i, m := range a.Models {
		models[i] = m.ID
	}

	permissions := a.PermissionModes
	if permissions == nil {
		permissions = []string{}
	}

	return domain.ManifestAgent{
		Installed:      a.Installed,
		Version:        a.VersionDetected,
		AuthValid:      authValid,
		Functional:     isFunctional,
		CooldownActive: isCooldown,
		ErrorClass:     errClass,
		Models:         models,
		Permissions:    permissions,
	}, isFunctional, isCooldown
}

func buildRolesSummary(resolved map[domain.Role]domain.RoleResolution) map[domain.Role]domain.ManifestResolvedRole {
	standardRoles := []domain.Role{
		domain.RolePrimary,
		domain.RoleSubagent,
		domain.RoleAuxiliary,
		domain.RoleReviewer,
	}

	out := make(map[domain.Role]domain.ManifestResolvedRole, len(standardRoles))
	for _, role := range standardRoles {
		res, ok := resolved[role]
		out[role] = formatResolvedRole(res, ok)
	}

	for role, res := range resolved {
		if _, exists := out[role]; !exists {
			out[role] = formatResolvedRole(res, true)
		}
	}
	return out
}

func formatResolvedRole(res domain.RoleResolution, ok bool) domain.ManifestResolvedRole {
	if !ok {
		return domain.ManifestResolvedRole{
			Agent:  defaultNoneValue,
			Model:  defaultNoneValue,
			Vendor: defaultNoneValue,
			Reason: defaultUnconfigured,
		}
	}

	agent := res.SelectedAgentID
	if agent == "" {
		agent = defaultNoneValue
	}
	model := res.SelectedModel
	if model == "" {
		model = defaultNoneValue
	}
	vendor := res.Vendor
	if vendor == "" {
		vendor = defaultNoneValue
	}
	reason := res.Reasoning
	if reason == "" {
		reason = defaultUnconfigured
	}

	return domain.ManifestResolvedRole{
		Agent:  agent,
		Model:  model,
		Vendor: vendor,
		Reason: reason,
	}
}

func formatGeneratedAt(now time.Time) string {
	return now.UTC().Format(isoTimestampFormat)
}

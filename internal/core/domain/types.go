// Package domain provides pure immutable domain value types for Soundcheck.
package domain

// Tier represents the probe evaluation level.
type Tier string

// ProbeLevel is an alias for Tier.
type ProbeLevel = Tier

// Tier levels supported by probe execution.
const (
	TierL0 Tier = "L0"
	TierL1 Tier = "L1"
	TierL2 Tier = "L2"
	TierL3 Tier = "L3"
)

// ErrorClass represents the standardized error classification.
type ErrorClass string

// Standard error classes for classification and probes.
const (
	ErrorClassOK               ErrorClass = "ok"
	ErrorClassNotInstalled     ErrorClass = "not_installed"
	ErrorClassUnauthenticated  ErrorClass = "unauthenticated"
	ErrorClassRateLimited      ErrorClass = "rate_limited"
	ErrorClassQuotaExhausted   ErrorClass = "quota_exhausted"
	ErrorClassModelUnavailable ErrorClass = "model_unavailable"
	ErrorClassPermissionDenied ErrorClass = "permission_denied"
	ErrorClassTimeout          ErrorClass = "timeout"
	ErrorClassCorruptedState   ErrorClass = "corrupted_state"
)

// Classification represents the outcome of analyzing raw execution output.
type Classification struct {
	ErrorClass ErrorClass `json:"errorClass"`
	Reason     string     `json:"reason"`
	CaughtLie  bool       `json:"caughtLie"`
	Rewritten  int        `json:"rewritten,omitempty"`
	BackoffMin int        `json:"backoffMin,omitempty"`
	QuirkID    string     `json:"quirkId,omitempty"`
}

// RawOutput contains raw execution outputs for classification.
type RawOutput struct {
	AgentID    string `json:"agentId,omitempty"`
	ExitCode   int    `json:"exitCode"`
	Stdout     string `json:"stdout"`
	Stderr     string `json:"stderr"`
	ParsedJSON any    `json:"parsedJson,omitempty"`
}

// Quirk defines a documented agent CLI anomaly and its defense/classification rule.
type Quirk struct {
	ID                string     `json:"id" yaml:"id"`
	AgentID           string     `json:"agentId,omitempty" yaml:"agentId,omitempty"`
	Name              string     `json:"name,omitempty" yaml:"name,omitempty"`
	DocIssue          string     `json:"docIssue,omitempty" yaml:"docIssue,omitempty"`
	Category          string     `json:"category,omitempty" yaml:"category,omitempty"`
	Description       string     `json:"description,omitempty" yaml:"description,omitempty"`
	ReferenceURL      string     `json:"referenceUrl,omitempty" yaml:"referenceUrl,omitempty"`
	DiscoveredDate    string     `json:"discoveredDate,omitempty" yaml:"discoveredDate,omitempty"`
	Severity          string     `json:"severity,omitempty" yaml:"severity,omitempty"`
	Impact            string     `json:"impact,omitempty" yaml:"impact,omitempty"`
	Resolution        string     `json:"resolution,omitempty" yaml:"resolution,omitempty"`
	TriggerExitCode   *int       `json:"triggerExitCode,omitempty" yaml:"triggerExitCode,omitempty"`
	TriggerRegex      string     `json:"triggerRegex,omitempty" yaml:"triggerRegex,omitempty"`
	ErrorClass        ErrorClass `json:"errorClass,omitempty" yaml:"errorClass,omitempty"`
	RewrittenExitCode int        `json:"rewrittenExitCode,omitempty" yaml:"rewrittenExitCode,omitempty"`
	CooldownMinutes   int        `json:"cooldownMinutes,omitempty" yaml:"cooldownMinutes,omitempty"`
	CaughtLie         bool       `json:"caughtLie,omitempty" yaml:"caughtLie,omitempty"`
}

// ModelInfo describes a supported model for an agent.
type ModelInfo struct {
	ID             string  `json:"id" yaml:"id"`
	DisplayName    string  `json:"displayName" yaml:"displayName"`
	Vendor         string  `json:"vendor" yaml:"vendor"`
	Tier           string  `json:"tier" yaml:"tier"`
	ContextWindow  int     `json:"contextWindow" yaml:"contextWindow"`
	InputCostPerM  float64 `json:"inputCostPerM" yaml:"inputCostPerM"`
	OutputCostPerM float64 `json:"outputCostPerM" yaml:"outputCostPerM"`
	Verified       bool    `json:"verified" yaml:"verified"`
}

// Probe specifies the execution parameters for a tier probe.
type Probe struct {
	Level            Tier   `json:"level" yaml:"level"`
	Command          string `json:"command" yaml:"command"`
	TimeoutMs        int    `json:"timeoutMs,omitempty" yaml:"timeoutMs,omitempty"`
	ExpectedExitCode int    `json:"expectedExitCode,omitempty" yaml:"expectedExitCode,omitempty"`
}

// RawDetails contains the raw execution output attached to a probe result.
type RawDetails struct {
	ExitCode       int    `json:"exitCode,omitempty"`
	Stdout         string `json:"stdout,omitempty"`
	Stderr         string `json:"stderr,omitempty"`
	ParsedJSON     any    `json:"parsedJson,omitempty"`
	Simulated      bool   `json:"simulated,omitempty"`
	QuirkTriggered string `json:"quirkTriggered,omitempty"`
}

// ProbeResult captures the outcome of evaluating an agent at a given tier.
type ProbeResult struct {
	Level       Tier        `json:"level"`
	Status      string      `json:"status"`
	ErrorClass  ErrorClass  `json:"errorClass"`
	Message     string      `json:"message"`
	BackoffMin  int         `json:"backoffMin,omitempty"`
	Timestamp   int64       `json:"timestamp,omitempty"`
	CachedUntil int64       `json:"cachedUntil,omitempty"`
	LatencyMs   float64     `json:"latencyMs,omitempty"`
	RawDetails  *RawDetails `json:"rawDetails,omitempty"`
}

// Adapter defines the configuration, quirks, models, and probe status for an agent CLI.
type Adapter struct {
	ID                  string               `json:"id" yaml:"id"`
	Name                string               `json:"name" yaml:"name"`
	Vendor              string               `json:"vendor" yaml:"vendor"`
	Binary              string               `json:"binary" yaml:"binary"`
	Category            string               `json:"category" yaml:"category"`
	VersionSupported    string               `json:"versionSupported,omitempty" yaml:"versionSupported,omitempty"`
	Installed           bool                 `json:"installed" yaml:"installed"`
	VersionDetected     string               `json:"versionDetected,omitempty" yaml:"versionDetected,omitempty"`
	AuthMethodSupported []string             `json:"authMethodSupported,omitempty" yaml:"authMethodSupported,omitempty"`
	HeadlessCommand     string               `json:"headlessCommand,omitempty" yaml:"headlessCommand,omitempty"`
	PermissionModes     []string             `json:"permissionModes,omitempty" yaml:"permissionModes,omitempty"`
	Quirks              []Quirk              `json:"quirks,omitempty" yaml:"quirks,omitempty"`
	Models              []ModelInfo          `json:"models,omitempty" yaml:"models,omitempty"`
	Probes              map[Tier]ProbeResult `json:"probes,omitempty" yaml:"probes,omitempty"`
	CooldownUntil       *int64               `json:"cooldownUntil,omitempty" yaml:"cooldownUntil,omitempty"`
	AllowedFails        int                  `json:"allowedFails,omitempty" yaml:"allowedFails,omitempty"`
	ConsecutiveFails    int                  `json:"consecutiveFails,omitempty" yaml:"consecutiveFails,omitempty"`
}

// Role defines an agent execution assignment role.
type Role string

// Supported agent roles for resolution.
const (
	RolePrimary   Role = "primary"
	RoleSubagent  Role = "subagent"
	RoleAuxiliary Role = "auxiliary"
	RoleReviewer  Role = "reviewer"
)

// StepStatus indicates the resolution status of an evaluation step.
type StepStatus string

// Step resolution statuses.
const (
	StepStatusSelected              StepStatus = "selected"
	StepStatusSkippedCooldown       StepStatus = "skipped_cooldown"
	StepStatusSkippedUninstalled    StepStatus = "skipped_uninstalled"
	StepStatusSkippedUnauth         StepStatus = "skipped_unauth"
	StepStatusSkippedVendorConflict StepStatus = "skipped_vendor_conflict"
)

// ResolverStep logs a candidate evaluation during role resolution.
type ResolverStep struct {
	Candidate string     `json:"candidate"`
	AgentID   string     `json:"agentId"`
	ModelID   string     `json:"modelId"`
	Status    StepStatus `json:"status"`
	Details   string     `json:"details"`
}

// Assignment captures the resolved agent and model for a given role.
type Assignment struct {
	Role              Role           `json:"role"`
	SelectedAgentID   string         `json:"selectedAgentId"`
	SelectedModel     string         `json:"selectedModel"`
	FallbackChain     []string       `json:"fallbackChain"`
	Reasoning         string         `json:"reasoning"`
	Vendor            string         `json:"vendor"`
	EstimatedCostPerM float64        `json:"estimatedCostPerM"`
	Healthy           bool           `json:"healthy"`
	Steps             []ResolverStep `json:"steps,omitempty"`
}

// ManifestHost describes the host execution environment in a capability manifest.
type ManifestHost struct {
	Platform    string `json:"platform"`
	NodeVersion string `json:"nodeVersion,omitempty"`
	GitWorktree bool   `json:"gitWorktree"`
}

// ManifestSummary summarizes agent health and defenses in a capability manifest.
type ManifestSummary struct {
	TotalAgents      int `json:"totalAgents"`
	FunctionalAgents int `json:"functionalAgents"`
	CooldownsActive  int `json:"cooldownsActive"`
	QuirksProtected  int `json:"quirksProtected"`
}

// ManifestAgent summarizes an agent's capability state in a manifest.
type ManifestAgent struct {
	Installed      bool       `json:"installed"`
	Version        string     `json:"version,omitempty"`
	AuthValid      bool       `json:"authValid"`
	Functional     bool       `json:"functional"`
	CooldownActive bool       `json:"cooldownActive"`
	ErrorClass     ErrorClass `json:"errorClass"`
	Models         []string   `json:"models"`
	Permissions    []string   `json:"permissions"`
}

// ManifestResolvedRole summarizes a resolved role assignment in a manifest.
type ManifestResolvedRole struct {
	Agent  string `json:"agent"`
	Model  string `json:"model"`
	Vendor string `json:"vendor"`
	Reason string `json:"reason"`
}

// Manifest captures the complete capability manifest payload.
type Manifest struct {
	SchemaVersion string                        `json:"schemaVersion"`
	GeneratedAt   string                        `json:"generatedAt"`
	Host          ManifestHost                  `json:"host"`
	Summary       ManifestSummary               `json:"summary"`
	Agents        map[string]ManifestAgent      `json:"agents"`
	ResolvedRoles map[Role]ManifestResolvedRole `json:"resolvedRoles"`
}

// ProbePolicy configures probe behavior.
type ProbePolicy struct {
	Deep       bool    `json:"deep" yaml:"deep"`
	TTLMinutes float64 `json:"ttl_minutes" yaml:"ttl_minutes"`
}

// Policy specifies orchestrator policy constraints and thresholds.
type Policy struct {
	RequireDistinctVendor []Role      `json:"require_distinct_vendor" yaml:"require_distinct_vendor"`
	MaxCostUSDPerRun      float64     `json:"max_cost_usd_per_run,omitempty" yaml:"max_cost_usd_per_run,omitempty"`
	Probe                 ProbePolicy `json:"probe,omitempty" yaml:"probe,omitempty"`
}

// Config represents the top-level Soundcheck configuration (agent-doctor.yaml).
type Config struct {
	Version int               `json:"version" yaml:"version"`
	Roles   map[Role][]string `json:"roles" yaml:"roles"`
	Policy  Policy            `json:"policy" yaml:"policy"`
}

// ResolverExplanationStep aliases ResolverStep for compatibility.
type ResolverExplanationStep = ResolverStep

// DetailedResolution aliases Assignment for compatibility.
type DetailedResolution = Assignment

// RoleResolution aliases Assignment for role resolution results.
type RoleResolution = Assignment

// HostInfo aliases ManifestHost for host execution environment info.
type HostInfo = ManifestHost

// Gate defines deterministic verification gates for a workflow stage.
type Gate struct {
	Tests        string `json:"tests,omitempty" yaml:"tests,omitempty"`
	DiffNonEmpty bool   `json:"diff_nonempty,omitempty" yaml:"diff_nonempty,omitempty"`
	MaxLoops     int    `json:"max_loops,omitempty" yaml:"max_loops,omitempty"`
}

// WorkflowStage represents a single node in a workflow DAG.
type WorkflowStage struct {
	ID             string   `json:"id" yaml:"id"`
	Name           string   `json:"name" yaml:"name"`
	Role           Role     `json:"role" yaml:"role"`
	Needs          []string `json:"needs,omitempty" yaml:"needs,omitempty"`
	Permissions    string   `json:"permissions" yaml:"permissions"`
	Worktree       bool     `json:"worktree,omitempty" yaml:"worktree,omitempty"`
	OutputArtifact string   `json:"outputArtifact,omitempty" yaml:"outputArtifact,omitempty"`
	Gate           *Gate    `json:"gate,omitempty" yaml:"gate,omitempty"`
}

// WorkflowPattern defines a declarative multi-stage agent pipeline DAG.
type WorkflowPattern struct {
	ID          string          `json:"id" yaml:"id"`
	Name        string          `json:"name" yaml:"name"`
	Description string          `json:"description" yaml:"description"`
	Stages      []WorkflowStage `json:"stages" yaml:"stages"`
	RawYAML     string          `json:"rawYaml,omitempty" yaml:"rawYaml,omitempty"`
}

// GateInput contains runtime execution artifacts to be evaluated by a stage gate.
type GateInput struct {
	HasDiff      bool   `json:"hasDiff"`
	DiffSize     int64  `json:"diffSize"`
	TestExitCode int    `json:"testExitCode"`
	TestPassed   bool   `json:"testPassed"`
	TestOutput   string `json:"testOutput,omitempty"`
	CurrentLoop  int    `json:"currentLoop"`
}

// GateEvaluation captures the outcome of evaluating stage gate conditions.
type GateEvaluation struct {
	Passed     bool     `json:"passed"`
	DiffPassed bool     `json:"diffPassed"`
	TestPassed bool     `json:"testPassed"`
	LoopPassed bool     `json:"loopPassed"`
	Failures   []string `json:"failures,omitempty"`
	Message    string   `json:"message"`
}

// PreflightEvaluation captures the result of verifying agent health before stage execution.
type PreflightEvaluation struct {
	Passed   bool     `json:"passed"`
	Failures []string `json:"failures,omitempty"`
	Message  string   `json:"message"`
}

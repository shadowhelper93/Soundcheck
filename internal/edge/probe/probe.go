package probe

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/soundcheck/soundcheck/internal/adapters"
	"github.com/soundcheck/soundcheck/internal/core/classify"
	"github.com/soundcheck/soundcheck/internal/core/domain"
	"github.com/soundcheck/soundcheck/internal/edge/exec"
	"github.com/soundcheck/soundcheck/internal/edge/platform"
	"github.com/soundcheck/soundcheck/internal/edge/store"
)

// ProbeFn evaluates a single probe tier for an adapter.
//
//nolint:revive // Explicit name required by specification.
type ProbeFn func(ctx context.Context, runner exec.Runner, adapter domain.Adapter, spec domain.Probe) domain.ProbeResult

// Prober runs and caches tier probes across agent CLI adapters.
type Prober struct {
	runner  exec.Runner
	cache   *store.TTLCache
	pool    *Pool
	ttl     time.Duration
	maxTier domain.Tier
	deep    bool
}

// ProberOption configures Prober settings.
type ProberOption func(*Prober)

// WithRunner sets the command execution runner.
func WithRunner(r exec.Runner) ProberOption {
	return func(p *Prober) { p.runner = r }
}

// WithCache sets the TTLCache instance.
func WithCache(c *store.TTLCache) ProberOption {
	return func(p *Prober) { p.cache = c }
}

// WithWorkers sets the bounded worker pool size.
func WithWorkers(w int) ProberOption {
	return func(p *Prober) { p.pool = NewPool(w) }
}

// WithTTL sets probe result cache TTL duration.
func WithTTL(ttl time.Duration) ProberOption {
	return func(p *Prober) { p.ttl = ttl }
}

// WithMaxTier sets the highest tier level to evaluate.
func WithMaxTier(tier domain.Tier) ProberOption {
	return func(p *Prober) { p.maxTier = tier }
}

// WithDeep enables deep probing up to L3.
func WithDeep(deep bool) ProberOption {
	return func(p *Prober) { p.deep = deep }
}

// New creates a new Prober configured with functional options.
func New(opts ...ProberOption) *Prober {
	p := &Prober{
		runner:  exec.NewPipeRunner(),
		pool:    NewPool(0),
		ttl:     30 * time.Minute,
		maxTier: domain.TierL3,
	}
	for _, opt := range opts {
		opt(p)
	}
	return p
}

type probeCallConfig struct {
	maxTier   domain.Tier
	skipCache bool
}

// ProbeOption configures a single probe invocation.
//
//nolint:revive // Explicit name required by specification.
type ProbeOption func(*probeCallConfig)

// WithCallMaxTier sets the tier cutoff for a single probe call.
func WithCallMaxTier(tier domain.Tier) ProbeOption {
	return func(c *probeCallConfig) { c.maxTier = tier }
}

// WithCallSkipCache forces live probing by skipping cache reads.
func WithCallSkipCache(skip bool) ProbeOption {
	return func(c *probeCallConfig) { c.skipCache = skip }
}

func (p *Prober) resolveCallConfig(opts []ProbeOption) probeCallConfig {
	cfg := probeCallConfig{maxTier: p.maxTier}
	for _, opt := range opts {
		opt(&cfg)
	}
	return cfg
}

// ProbeL0 checks binary presence via LookPath and inspects detected version.
//
//nolint:revive // Explicit name required by specification.
func ProbeL0(ctx context.Context, runner exec.Runner, adapter domain.Adapter, spec domain.Probe) domain.ProbeResult {
	start := time.Now()
	binPath, err := platform.LookPath(adapter.Binary)
	if err != nil {
		return domain.ProbeResult{
			Level:      domain.TierL0,
			Status:     "warning",
			ErrorClass: domain.ErrorClassNotInstalled,
			Message:    fmt.Sprintf("Binary %q not found in PATH", adapter.Binary),
			LatencyMs:  float64(time.Since(start).Milliseconds()),
			Timestamp:  time.Now().UnixMilli(),
		}
	}

	timeout := defaultTimeout(spec.TimeoutMs, 3000)
	cmd := buildCmd(binPath, spec.Command, []string{"--version"}, timeout)
	res, runErr := runner.Run(ctx, cmd)
	latency := float64(time.Since(start).Milliseconds())

	if runErr != nil || res.ExitCode != 0 {
		return domain.ProbeResult{
			Level:      domain.TierL0,
			Status:     "passed",
			ErrorClass: domain.ErrorClassOK,
			Message:    fmt.Sprintf("Binary found at %s (version check skipped)", binPath),
			LatencyMs:  latency,
			Timestamp:  time.Now().UnixMilli(),
			RawDetails: makeRawDetails(res, ""),
		}
	}

	version := extractVersion(string(res.Stdout))
	return domain.ProbeResult{
		Level:      domain.TierL0,
		Status:     "passed",
		ErrorClass: domain.ErrorClassOK,
		Message:    fmt.Sprintf("Binary found at %s (%s)", binPath, version),
		LatencyMs:  latency,
		Timestamp:  time.Now().UnixMilli(),
		RawDetails: makeRawDetails(res, ""),
	}
}

// ProbeL1 performs an authentication check turn.
//
//nolint:revive // Explicit name required by specification.
func ProbeL1(ctx context.Context, runner exec.Runner, adapter domain.Adapter, spec domain.Probe) domain.ProbeResult {
	return runTurnProbe(ctx, runner, adapter, spec, domain.TierL1, []string{"--version"}, "Credentials verified")
}

// ProbeL2 executes a minimal smoke test command.
//
//nolint:revive // Explicit name required by specification.
func ProbeL2(ctx context.Context, runner exec.Runner, adapter domain.Adapter, spec domain.Probe) domain.ProbeResult {
	return runTurnProbe(ctx, runner, adapter, spec, domain.TierL2, []string{"--version"}, "Smoke test passed")
}

// ProbeL3 validates target model availability.
//
//nolint:revive // Explicit name required by specification.
func ProbeL3(ctx context.Context, runner exec.Runner, adapter domain.Adapter, spec domain.Probe) domain.ProbeResult {
	return runTurnProbe(ctx, runner, adapter, spec, domain.TierL3, []string{"--version"}, "Model verified")
}

func runTurnProbe(
	ctx context.Context,
	runner exec.Runner,
	adapter domain.Adapter,
	spec domain.Probe,
	tier domain.Tier,
	defaultArgs []string,
	passMsg string,
) domain.ProbeResult {
	start := time.Now()
	binPath, err := platform.LookPath(adapter.Binary)
	if err != nil {
		binPath = adapter.Binary
	}

	timeout := defaultTimeout(spec.TimeoutMs, 5000)
	cmd := buildCmd(binPath, spec.Command, defaultArgs, timeout)
	res, _ := runner.Run(ctx, cmd)
	latency := float64(time.Since(start).Milliseconds())

	raw := domain.RawOutput{
		AgentID:  adapter.ID,
		ExitCode: res.ExitCode,
		Stdout:   string(res.Stdout),
		Stderr:   string(res.Stderr),
	}
	classification := classify.Classify(raw, adapter.Quirks)

	return evaluateTurnResult(tier, res, classification, latency, passMsg)
}

func evaluateTurnResult(
	tier domain.Tier,
	res exec.Result,
	cl domain.Classification,
	latency float64,
	passMsg string,
) domain.ProbeResult {
	now := time.Now().UnixMilli()
	details := makeRawDetails(res, cl.QuirkID)

	if res.TimedOut {
		return domain.ProbeResult{
			Level:      tier,
			Status:     "failed",
			ErrorClass: domain.ErrorClassTimeout,
			Message:    "Probe execution timed out",
			LatencyMs:  latency,
			Timestamp:  now,
			RawDetails: details,
		}
	}

	effectiveCode := res.ExitCode
	if cl.CaughtLie && cl.Rewritten == 0 && cl.ErrorClass == domain.ErrorClassOK {
		effectiveCode = 0
	}

	if cl.ErrorClass != domain.ErrorClassOK || effectiveCode != 0 {
		msg := cl.Reason
		if msg == "" {
			msg = fmt.Sprintf("Command failed with exit code %d", res.ExitCode)
		}
		errClass := cl.ErrorClass
		if errClass == domain.ErrorClassOK {
			errClass = domain.ErrorClassCorruptedState
		}
		return domain.ProbeResult{
			Level:      tier,
			Status:     "failed",
			ErrorClass: errClass,
			Message:    msg,
			BackoffMin: cl.BackoffMin,
			LatencyMs:  latency,
			Timestamp:  now,
			RawDetails: details,
		}
	}

	return domain.ProbeResult{
		Level:      tier,
		Status:     "passed",
		ErrorClass: domain.ErrorClassOK,
		Message:    passMsg,
		LatencyMs:  latency,
		Timestamp:  now,
		RawDetails: details,
	}
}

// ProbeTier runs a single probe tier with optional TTL cache lookup.
func (p *Prober) ProbeTier(
	ctx context.Context,
	adapter domain.Adapter,
	tier domain.Tier,
	spec domain.Probe,
	skipCache bool,
) domain.ProbeResult {
	key := fmt.Sprintf("probe:%s:%s", adapter.ID, tier)

	if !skipCache && p.cache != nil {
		if data, ok, err := p.cache.Get(key); err == nil && ok {
			var cached domain.ProbeResult
			if err := json.Unmarshal(data, &cached); err == nil {
				return cached
			}
		}
	}

	var res domain.ProbeResult
	switch tier {
	case domain.TierL0:
		res = ProbeL0(ctx, p.runner, adapter, spec)
	case domain.TierL1:
		res = ProbeL1(ctx, p.runner, adapter, spec)
	case domain.TierL2:
		res = ProbeL2(ctx, p.runner, adapter, spec)
	case domain.TierL3:
		res = ProbeL3(ctx, p.runner, adapter, spec)
	}

	if p.cache != nil {
		if p.ttl > 0 && res.Status == "passed" {
			res.CachedUntil = time.Now().Add(p.ttl).UnixMilli()
			if data, err := json.Marshal(res); err == nil {
				_ = p.cache.Set(key, data, p.ttl)
			}
		} else if res.ErrorClass == domain.ErrorClassRateLimited || res.BackoffMin > 0 {
			backoff := res.BackoffMin
			if backoff <= 0 {
				backoff = 30
			}
			ttl := time.Duration(backoff) * time.Minute
			res.CachedUntil = time.Now().Add(ttl).UnixMilli()
			if data, err := json.Marshal(res); err == nil {
				_ = p.cache.Set(key, data, ttl)
			}
		}
	}

	return res
}

// ProbeAdapter evaluates all configured probe tiers for an adapter using stop-on-fail policy.
func (p *Prober) ProbeAdapter(ctx context.Context, def adapters.Definition, opts ...ProbeOption) domain.Adapter {
	cfg := p.resolveCallConfig(opts)
	ad := def.ToAdapter()
	tiers := []domain.Tier{domain.TierL0, domain.TierL1, domain.TierL2, domain.TierL3}

	for _, tier := range tiers {
		if tierIndex(tier) > tierIndex(cfg.maxTier) {
			if cached, ok := p.getCachedRateLimit(ad, tier); ok {
				ad.Probes[tier] = cached
				p.applyBackoffCooldown(&ad, cached)
				continue
			}
			ad.Probes[tier] = skippedProbe(tier)
			continue
		}

		spec, _ := def.ProbeSpec(tier)
		res := p.ProbeTier(ctx, ad, tier, spec, cfg.skipCache)
		ad.Probes[tier] = res

		if tier == domain.TierL0 {
			p.updateL0Details(&ad, res)
			p.applyLiveModels(ctx, &ad, def)
		}

		if res.Status != "passed" {
			p.applyBackoffCooldown(&ad, res)
			p.skipSubsequentTiers(&ad, tier, tiers)
			break
		}
	}

	return ad
}

func (p *Prober) skipSubsequentTiers(ad *domain.Adapter, currentTier domain.Tier, tiers []domain.Tier) {
	for _, t := range tiers {
		if tierIndex(t) > tierIndex(currentTier) {
			ad.Probes[t] = skippedProbe(t)
		}
	}
}

func (p *Prober) applyBackoffCooldown(ad *domain.Adapter, res domain.ProbeResult) {
	if res.ErrorClass == domain.ErrorClassRateLimited || res.BackoffMin > 0 {
		if res.CachedUntil > time.Now().UnixMilli() {
			ad.CooldownUntil = &res.CachedUntil
			return
		}
		backoff := res.BackoffMin
		if backoff <= 0 {
			backoff = 30
		}
		cooldown := time.Now().Add(time.Duration(backoff) * time.Minute).UnixMilli()
		ad.CooldownUntil = &cooldown
	}
}

func (p *Prober) getCachedRateLimit(adapter domain.Adapter, tier domain.Tier) (domain.ProbeResult, bool) {
	if p.cache == nil {
		return domain.ProbeResult{}, false
	}
	key := fmt.Sprintf("probe:%s:%s", adapter.ID, tier)
	data, ok, err := p.cache.Get(key)
	if err != nil || !ok {
		return domain.ProbeResult{}, false
	}
	var cached domain.ProbeResult
	if err := json.Unmarshal(data, &cached); err != nil {
		return domain.ProbeResult{}, false
	}
	if (cached.ErrorClass == domain.ErrorClassRateLimited || cached.BackoffMin > 0) && cached.CachedUntil > time.Now().UnixMilli() {
		return cached, true
	}
	return domain.ProbeResult{}, false
}

func (p *Prober) updateL0Details(ad *domain.Adapter, res domain.ProbeResult) {
	if res.Status == "passed" {
		ad.Installed = true
		if res.RawDetails != nil && res.RawDetails.Stdout != "" {
			ad.VersionDetected = extractVersion(res.RawDetails.Stdout)
		}
	} else {
		ad.Installed = false
	}
}

// ProbeAll sweeps all adapter definitions concurrently using the bounded worker pool.
func (p *Prober) ProbeAll(ctx context.Context, defs []adapters.Definition, opts ...ProbeOption) []domain.Adapter {
	if p.pool == nil {
		p.pool = NewPool(0)
	}
	results := make([]domain.Adapter, len(defs))
	tasks := make([]Task, len(defs))

	for i, d := range defs {
		idx := i
		def := d
		tasks[i] = func(taskCtx context.Context) error {
			results[idx] = p.ProbeAdapter(taskCtx, def, opts...)
			return nil
		}
	}

	_ = p.pool.Run(ctx, tasks)
	return results
}

func skippedProbe(tier domain.Tier) domain.ProbeResult {
	return domain.ProbeResult{
		Level:      tier,
		Status:     "pending",
		ErrorClass: domain.ErrorClassOK,
		Message:    "Skipped",
	}
}

func defaultTimeout(specMs, fallbackMs int) time.Duration {
	if specMs > 0 {
		return time.Duration(specMs) * time.Millisecond
	}
	return time.Duration(fallbackMs) * time.Millisecond
}

func extractVersion(stdout string) string {
	lines := strings.Split(strings.TrimSpace(stdout), "\n")
	if len(lines) > 0 && lines[0] != "" {
		return lines[0]
	}
	return "Detected"
}

func makeRawDetails(res exec.Result, quirkID string) *domain.RawDetails {
	return &domain.RawDetails{
		ExitCode:       res.ExitCode,
		Stdout:         string(res.Stdout),
		Stderr:         string(res.Stderr),
		QuirkTriggered: quirkID,
	}
}

func tierIndex(tier domain.Tier) int {
	switch tier {
	case domain.TierL0:
		return 0
	case domain.TierL1:
		return 1
	case domain.TierL2:
		return 2
	case domain.TierL3:
		return 3
	default:
		return -1
	}
}

func buildCmd(bin, cmdStr string, fallbackArgs []string, timeout time.Duration) exec.Cmd {
	targetBin := bin
	var args []string

	switch {
	case bin != "" && cmdStr == bin:
		targetBin = bin
		args = fallbackArgs
	case bin != "" && strings.HasPrefix(cmdStr, bin+" "):
		targetBin = bin
		rest := strings.TrimSpace(strings.TrimPrefix(cmdStr, bin))
		parsed := splitCommand(rest)
		if len(parsed) > 0 {
			args = parsed
		} else {
			args = fallbackArgs
		}
	default:
		tokens := splitCommand(cmdStr)
		args = fallbackArgs
		if len(tokens) > 0 {
			if lp, err := platform.LookPath(tokens[0]); err == nil {
				targetBin = lp
			} else if bin != "" {
				targetBin = bin
			} else {
				targetBin = tokens[0]
			}
			if len(tokens) > 1 {
				args = tokens[1:]
			}
		}
	}
	return exec.Cmd{
		Binary:  targetBin,
		Args:    args,
		Timeout: timeout,
	}
}

func splitCommand(cmd string) []string {
	var tokens []string
	var cur strings.Builder
	var inQuote rune
	var escaped bool

	for _, r := range cmd {
		if escaped {
			cur.WriteRune(r)
			escaped = false
			continue
		}
		if r == '\\' {
			escaped = true
			continue
		}
		if inQuote != 0 {
			if r == inQuote {
				inQuote = 0
			} else {
				cur.WriteRune(r)
			}
			continue
		}
		if r == '\'' || r == '"' {
			inQuote = r
			continue
		}
		if r == ' ' || r == '\t' || r == '\n' {
			if cur.Len() > 0 {
				tokens = append(tokens, cur.String())
				cur.Reset()
			}
			continue
		}
		cur.WriteRune(r)
	}
	if cur.Len() > 0 {
		tokens = append(tokens, cur.String())
	}
	return tokens
}

// applyLiveModels replaces the static catalog with the account's real models when the CLI can list them.
// Without discovery the catalog stays but is marked unverified.
func (p *Prober) applyLiveModels(ctx context.Context, ad *domain.Adapter, def adapters.Definition) {
	if !ad.Installed {
		return
	}
	binPath, err := platform.LookPath(ad.Binary)
	if err != nil {
		return
	}
	if live, ok := discoverModels(ctx, p.runner, def, binPath); ok {
		ad.Models = live
		return
	}
	models := make([]domain.ModelInfo, len(ad.Models))
	copy(models, ad.Models)
	for i := range models {
		models[i].Verified = false
	}
	ad.Models = models
}

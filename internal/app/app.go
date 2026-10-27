// Package app provides application use-case orchestration for Soundcheck.
package app

import (
	"context"
	"fmt"
	"os"
	"runtime"
	"strings"
	"time"

	"github.com/soundcheck/soundcheck/internal/adapters"
	"github.com/soundcheck/soundcheck/internal/core/domain"
	"github.com/soundcheck/soundcheck/internal/core/manifest"
	"github.com/soundcheck/soundcheck/internal/core/resolve"
	"github.com/soundcheck/soundcheck/internal/edge/exec"
	"github.com/soundcheck/soundcheck/internal/edge/probe"
	"github.com/soundcheck/soundcheck/internal/edge/store"
)

// App orchestrates CLI adapters, probers, config, and core engines.
type App struct {
	cfg      domain.Config
	runner   exec.Runner
	store    store.Store
	cache    *store.TTLCache
	prober   *probe.Prober
	defs     map[string]adapters.Definition
	defList  []adapters.Definition
	probeTTL time.Duration
}

// Option configures App settings.
type Option func(*App)

// WithConfig sets the domain configuration.
func WithConfig(cfg domain.Config) Option {
	return func(a *App) { a.cfg = cfg }
}

// WithRunner sets the process execution runner.
func WithRunner(r exec.Runner) Option {
	return func(a *App) { a.runner = r }
}

// WithStore sets the persistence store.
func WithStore(s store.Store) Option {
	return func(a *App) { a.store = s }
}

// WithCache sets the TTLCache instance.
func WithCache(c *store.TTLCache) Option {
	return func(a *App) { a.cache = c }
}

// WithProber sets a pre-configured Prober.
func WithProber(p *probe.Prober) Option {
	return func(a *App) { a.prober = p }
}

// WithDefinitions sets explicit adapter definitions.
func WithDefinitions(defs []adapters.Definition) Option {
	return func(a *App) {
		a.defList = defs
		a.defs = make(map[string]adapters.Definition, len(defs))
		for _, d := range defs {
			a.defs[d.ID] = d
		}
	}
}

// WithProbeTTL sets the default probe cache TTL.
func WithProbeTTL(ttl time.Duration) Option {
	return func(a *App) { a.probeTTL = ttl }
}

// New constructs an App orchestrator with functional options.
func New(opts ...Option) (*App, error) {
	defs := adapters.All()
	defMap := make(map[string]adapters.Definition, len(defs))
	for _, d := range defs {
		defMap[d.ID] = d
	}

	a := &App{
		cfg:      resolve.DefaultConfig(),
		runner:   exec.NewPipeRunner(),
		cache:    store.NewTTLCache(nil),
		defList:  defs,
		defs:     defMap,
		probeTTL: 30 * time.Minute,
	}

	for _, opt := range opts {
		opt(a)
	}

	if a.prober == nil {
		a.prober = probe.New(
			probe.WithRunner(a.runner),
			probe.WithCache(a.cache),
			probe.WithTTL(a.probeTTL),
		)
	}
	return a, nil
}

// ProbeOne executes full L0..L3 probes for a single adapter.
func (a *App) ProbeOne(ctx context.Context, adapterID string) (domain.Adapter, error) {
	def, ok := a.defs[adapterID]
	if !ok {
		return domain.Adapter{}, fmt.Errorf("unknown adapter: %s", adapterID)
	}
	return a.prober.ProbeAdapter(ctx, def, probe.WithCallMaxTier(domain.TierL3)), nil
}

// Resolve evaluates role assignments against current adapter capabilities.
func (a *App) Resolve(ctx context.Context, roleFilter string) (map[domain.Role]domain.DetailedResolution, error) {
	probed := a.prober.ProbeAll(ctx, a.defList)
	res, err := resolve.Roles(a.cfg, probed, time.Now())
	if err != nil {
		return nil, err
	}
	if roleFilter == "" {
		return res, nil
	}
	targetRole := domain.Role(roleFilter)
	filtered := make(map[domain.Role]domain.DetailedResolution)
	if val, ok := res[targetRole]; ok {
		filtered[targetRole] = val
	}
	return filtered, nil
}

// Manifest builds the complete capability manifest.
func (a *App) Manifest(ctx context.Context) (domain.Manifest, error) {
	report, err := a.Doctor(ctx)
	if err != nil {
		return domain.Manifest{}, err
	}
	return manifest.BuildManifest(report.Adapters, report.Resolutions, report.Host, report.Timestamp), nil
}

// HostInfo inspects the host runtime environment.
func (a *App) HostInfo(ctx context.Context) domain.HostInfo {
	return domain.HostInfo{
		Platform:    runtime.GOOS,
		NodeVersion: detectNodeVersion(ctx, a.runner),
		GitWorktree: detectGitWorktree(ctx, a.runner),
	}
}

func detectNodeVersion(ctx context.Context, runner exec.Runner) string {
	res, err := runner.Run(ctx, exec.Cmd{
		Binary:  "node",
		Args:    []string{"--version"},
		Timeout: 2 * time.Second,
	})
	if err == nil && res.ExitCode == 0 {
		return strings.TrimSpace(string(res.Stdout))
	}
	return ""
}

func detectGitWorktree(ctx context.Context, runner exec.Runner) bool {
	res, err := runner.Run(ctx, exec.Cmd{
		Binary:  "git",
		Args:    []string{"rev-parse", "--is-inside-work-tree"},
		Timeout: 2 * time.Second,
	})
	if err == nil && res.ExitCode == 0 && strings.TrimSpace(string(res.Stdout)) == "true" {
		return true
	}
	fi, statErr := os.Stat(".git")
	return statErr == nil && (fi.IsDir() || fi.Mode().IsRegular())
}

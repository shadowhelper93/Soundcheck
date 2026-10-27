package app

import (
	"context"
	"fmt"
	"time"

	"github.com/soundcheck/soundcheck/internal/core/domain"
	"github.com/soundcheck/soundcheck/internal/core/manifest"
	"github.com/soundcheck/soundcheck/internal/core/resolve"
	"github.com/soundcheck/soundcheck/internal/edge/probe"
)

// DoctorReport contains the aggregated doctor status, probed adapters, and role assignments.
type DoctorReport struct {
	Timestamp   time.Time                                 `json:"timestamp"`
	Adapters    []domain.Adapter                          `json:"adapters"`
	Resolutions map[domain.Role]domain.DetailedResolution `json:"resolutions"`
	Summary     domain.ManifestSummary                    `json:"summary"`
	Host        domain.HostInfo                           `json:"host"`
}

type doctorOptions struct {
	deep      bool
	forceLive bool
}

// DoctorOption configures Doctor execution.
type DoctorOption func(*doctorOptions)

// WithDoctorDeep forces deep probe execution through L3.
func WithDoctorDeep(deep bool) DoctorOption {
	return func(o *doctorOptions) { o.deep = deep }
}

// WithDoctorForce forces live probe execution, bypassing cached probe results.
func WithDoctorForce(force bool) DoctorOption {
	return func(o *doctorOptions) { o.forceLive = force }
}

// Doctor runs probes across adapters (honoring TTL), resolves roles, and outputs status.
func (a *App) Doctor(ctx context.Context, opts ...DoctorOption) (DoctorReport, error) {
	var dopts doctorOptions
	for _, opt := range opts {
		opt(&dopts)
	}

	cfg := a.Config()
	var probeOpts []probe.ProbeOption
	if dopts.deep || cfg.Policy.Probe.Deep {
		probeOpts = append(probeOpts, probe.WithCallMaxTier(domain.TierL3))
	}
	if dopts.forceLive {
		probeOpts = append(probeOpts, probe.WithCallSkipCache(true))
	}

	probed := a.prober.ProbeAll(ctx, a.defList, probeOpts...)
	now := time.Now()

	resolutions, err := resolve.Roles(cfg, probed, now)
	if err != nil {
		return DoctorReport{}, fmt.Errorf("resolve roles: %w", err)
	}

	host := a.HostInfo(ctx)
	m := manifest.BuildManifest(probed, resolutions, host, now)

	return DoctorReport{
		Timestamp:   now,
		Adapters:    probed,
		Resolutions: resolutions,
		Summary:     m.Summary,
		Host:        host,
	}, nil
}

package cli

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/spf13/cobra"

	"github.com/soundcheck/soundcheck/internal/app"
	"github.com/soundcheck/soundcheck/internal/core/domain"
)

func newProbeCmd(opts *RootOptions, injected *app.App) *cobra.Command {
	return &cobra.Command{
		Use:   "probe [adapter]",
		Short: "Run probe for one or all adapters",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, cancel := resolveContext(cmd.Context(), opts.Timeout)
			defer cancel()
			cmd.SetContext(ctx)

			a, err := resolveApp(opts, injected)
			if err != nil {
				return err
			}

			if len(args) == 1 {
				return runProbeOne(cmd, opts, a, args[0])
			}
			return runProbeAll(cmd, opts, a)
		},
	}
}

func runProbeOne(cmd *cobra.Command, opts *RootOptions, a *app.App, id string) error {
	adapter, err := a.ProbeOne(cmd.Context(), id)
	if err != nil {
		return &ExitError{Code: ExitUsage, Err: err}
	}
	if err := outputProbeAdapter(cmd.OutOrStdout(), opts, adapter); err != nil {
		return err
	}
	return probeExitCode(adapter)
}

func runProbeAll(cmd *cobra.Command, opts *RootOptions, a *app.App) error {
	report, err := a.Doctor(cmd.Context(), app.WithDoctorDeep(true))
	if err != nil {
		return &ExitError{Code: ExitFailed, Err: err}
	}
	if err := outputProbeList(cmd.OutOrStdout(), opts, report.Adapters); err != nil {
		return err
	}
	return allProbesExitCode(report.Adapters)
}

func outputProbeAdapter(w io.Writer, opts *RootOptions, ad domain.Adapter) error {
	if opts.JSON {
		b, err := json.MarshalIndent(ad, "", "  ")
		if err != nil {
			return &ExitError{Code: ExitFailed, Err: err}
		}
		_, err = fmt.Fprintln(w, string(b))
		return err
	}
	if opts.NDJSON {
		enc := json.NewEncoder(w)
		return enc.Encode(map[string]any{"type": "probe", "adapter": ad})
	}
	return renderSingleAdapter(w, ad, isColorDisabled(opts))
}

func outputProbeList(w io.Writer, opts *RootOptions, ads []domain.Adapter) error {
	if opts.JSON {
		b, err := json.MarshalIndent(ads, "", "  ")
		if err != nil {
			return &ExitError{Code: ExitFailed, Err: err}
		}
		_, err = fmt.Fprintln(w, string(b))
		return err
	}
	if opts.NDJSON {
		enc := json.NewEncoder(w)
		for _, ad := range ads {
			if err := enc.Encode(map[string]any{"type": "probe", "adapter": ad}); err != nil {
				return err
			}
		}
		return nil
	}
	return renderAdapterList(w, ads, isColorDisabled(opts))
}

func probeExitCode(ad domain.Adapter) error {
	if !ad.Installed {
		return &ExitError{Code: ExitFailed, Err: fmt.Errorf("adapter not installed: %s", ad.ID)}
	}
	if !isAdapterFunctional(ad) {
		return &ExitError{Code: ExitDegraded, Err: fmt.Errorf("adapter degraded: %s", ad.ID)}
	}
	return nil
}

func allProbesExitCode(ads []domain.Adapter) error {
	functional := 0
	for _, ad := range ads {
		if isAdapterFunctional(ad) {
			functional++
		}
	}
	if functional == 0 && len(ads) > 0 {
		return &ExitError{Code: ExitFailed}
	}
	if functional < len(ads) {
		return &ExitError{Code: ExitDegraded}
	}
	return nil
}

func renderSingleAdapter(w io.Writer, ad domain.Adapter, noColor bool) error {
	status := adapterStatus(ad, noColor)
	meter := formatTierMeter(ad.Probes, noColor)
	ver := ad.VersionDetected
	if ver == "" {
		ver = "n/a"
	}
	if _, err := fmt.Fprintf(w, "%s (%s) [%s]\n", ad.Name, ad.ID, status); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "Version: %s · Tier Meter: %s\n", ver, meter); err != nil {
		return err
	}
	tiers := []domain.Tier{domain.TierL0, domain.TierL1, domain.TierL2, domain.TierL3}
	for _, t := range tiers {
		if p, ok := ad.Probes[t]; ok {
			if _, err := fmt.Fprintf(w, "  %-4s: %-8s (latency %.0fms) - %s\n", t, p.Status, p.LatencyMs, p.Message); err != nil {
				return err
			}
		}
	}
	return nil
}

func renderAdapterList(w io.Writer, ads []domain.Adapter, noColor bool) error {
	for _, ad := range ads {
		meter := formatTierMeter(ad.Probes, noColor)
		status := adapterStatus(ad, noColor)
		ver := ad.VersionDetected
		if ver == "" {
			ver = "n/a"
		}
		if _, err := fmt.Fprintf(w, "%-10s %-20s %-12s %s  %s\n", ad.ID, ad.Name, ver, meter, status); err != nil {
			return err
		}
	}
	return nil
}

package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/soundcheck/soundcheck/internal/app"
	"github.com/soundcheck/soundcheck/internal/core/domain"
)

func newDoctorCmd(opts *RootOptions, injected *app.App) *cobra.Command {
	var deep, force bool
	cmd := &cobra.Command{
		Use:   "doctor",
		Short: "Run system and CLI adapter health preflight check",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			ctx, cancel := resolveContext(cmd.Context(), opts.Timeout)
			defer cancel()

			a, err := resolveApp(opts, injected)
			if err != nil {
				return err
			}

			dopts := buildDoctorOptions(deep, force)
			report, err := a.Doctor(ctx, dopts...)
			if err != nil {
				return &ExitError{Code: ExitFailed, Err: err}
			}

			if err := outputDoctor(cmd.OutOrStdout(), opts, report); err != nil {
				return err
			}
			return doctorExitCode(report)
		},
	}
	cmd.Flags().BoolVar(&deep, "deep", false, "Force deep probe execution through L3")
	cmd.Flags().BoolVar(&force, "force", false, "Bypass cache and force live probes")
	return cmd
}

func buildDoctorOptions(deep, force bool) []app.DoctorOption {
	var dopts []app.DoctorOption
	if deep {
		dopts = append(dopts, app.WithDoctorDeep(true))
	}
	if force {
		dopts = append(dopts, app.WithDoctorForce(true))
	}
	return dopts
}

func doctorExitCode(r app.DoctorReport) error {
	if r.Summary.FunctionalAgents == 0 && r.Summary.TotalAgents > 0 {
		return &ExitError{Code: ExitFailed}
	}
	if r.Summary.FunctionalAgents < r.Summary.TotalAgents || r.Summary.CooldownsActive > 0 {
		return &ExitError{Code: ExitDegraded}
	}
	for _, res := range r.Resolutions {
		if !res.Healthy || res.SelectedAgentID == "" {
			return &ExitError{Code: ExitDegraded}
		}
	}
	return nil
}

func outputDoctor(w io.Writer, opts *RootOptions, r app.DoctorReport) error {
	if opts.JSON {
		b, err := json.MarshalIndent(r, "", "  ")
		if err != nil {
			return &ExitError{Code: ExitFailed, Err: err}
		}
		_, err = fmt.Fprintln(w, string(b))
		return err
	}
	if opts.NDJSON {
		return outputDoctorNDJSON(w, r)
	}
	return renderDoctorTable(w, r, isColorDisabled(opts))
}

func outputDoctorNDJSON(w io.Writer, r app.DoctorReport) error {
	enc := json.NewEncoder(w)
	if err := enc.Encode(map[string]any{"type": "host", "data": r.Host}); err != nil {
		return err
	}
	for _, ad := range r.Adapters {
		if err := enc.Encode(map[string]any{"type": "adapter", "data": ad}); err != nil {
			return err
		}
	}
	for role, res := range r.Resolutions {
		if err := enc.Encode(map[string]any{"type": "resolution", "role": role, "data": res}); err != nil {
			return err
		}
	}
	return enc.Encode(map[string]any{"type": "summary", "data": r.Summary})
}

func renderDoctorTable(w io.Writer, r app.DoctorReport, noColor bool) error {
	if err := renderHeader(w, r, noColor); err != nil {
		return err
	}
	if _, err := fmt.Fprintln(w, "ADAPTERS"); err != nil {
		return err
	}
	for _, ad := range r.Adapters {
		meter := formatTierMeter(ad.Probes, noColor)
		status := adapterStatus(ad, noColor)
		ver := ad.VersionDetected
		if ver == "" {
			ver = "n/a"
		}
		if _, err := fmt.Fprintf(w, "  %-10s %-20s %-12s %s  %s\n", ad.ID, ad.Name, ver, meter, status); err != nil {
			return err
		}
	}
	if _, err := fmt.Fprintln(w, "\nRESOLVED ROLES"); err != nil {
		return err
	}
	if err := renderDoctorResolutions(w, r.Resolutions, noColor); err != nil {
		return err
	}
	_, err := fmt.Fprintln(w)
	return err
}

func renderHeader(w io.Writer, r app.DoctorReport, noColor bool) error {
	title := "SOUNDCHECK PREFLIGHT REPORT"
	if !noColor {
		title = "\x1b[1m\x1b[32m✔ " + title + "\x1b[0m"
	}
	if _, err := fmt.Fprintln(w, title); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "OS: %s · Node: %s · Worktree: %t\n", r.Host.Platform, r.Host.NodeVersion, r.Host.GitWorktree); err != nil {
		return err
	}
	_, err := fmt.Fprintf(w, "Summary: %d/%d functional · %d cooldowns · %d quirks protected\n\n",
		r.Summary.FunctionalAgents, r.Summary.TotalAgents, r.Summary.CooldownsActive, r.Summary.QuirksProtected)
	return err
}

func renderDoctorResolutions(w io.Writer, res map[domain.Role]domain.DetailedResolution, noColor bool) error {
	roles := sortedRoles(res)
	for _, role := range roles {
		r := res[role]
		status := "HEALTHY"
		if !r.Healthy {
			status = "UNHEALTHY"
		}
		if !noColor {
			status = colorizeStatus(status)
		}
		if _, err := fmt.Fprintf(w, "  %-12s %-10s (%s) [%s]\n", role, r.SelectedAgentID, r.SelectedModel, status); err != nil {
			return err
		}
	}
	return nil
}

func tierGlyph(res domain.ProbeResult, ok bool) (string, string) {
	if !ok {
		return "-", "90"
	}
	switch res.Status {
	case "passed":
		return "✓", "32"
	case "failed":
		return "✖", "31"
	case "warn", "degraded":
		return "▲", "33"
	default:
		return "-", "90"
	}
}

func formatTierMeter(probes map[domain.Tier]domain.ProbeResult, noColor bool) string {
	tiers := []domain.Tier{domain.TierL0, domain.TierL1, domain.TierL2, domain.TierL3}
	parts := make([]string, len(tiers))
	for i, t := range tiers {
		res, ok := probes[t]
		glyph, color := tierGlyph(res, ok)
		if noColor {
			parts[i] = fmt.Sprintf("%s:%s", t, glyph)
		} else {
			parts[i] = fmt.Sprintf("%s:\x1b[%sm%s\x1b[0m", t, color, glyph)
		}
	}
	return "[" + strings.Join(parts, " ") + "]"
}

func isAdapterFunctional(ad domain.Adapter) bool {
	l1, hasL1 := ad.Probes[domain.TierL1]
	authValid := hasL1 && l1.Status == "passed"
	isCooldown := ad.CooldownUntil != nil && *ad.CooldownUntil > time.Now().UnixMilli()
	return ad.Installed && authValid && !isCooldown
}

func adapterStatus(ad domain.Adapter, noColor bool) string {
	if !ad.Installed {
		if noColor {
			return "NOT FOUND"
		}
		return "\x1b[31mNOT FOUND\x1b[0m"
	}
	if !isAdapterFunctional(ad) {
		if noColor {
			return "DEGRADED"
		}
		return "\x1b[33mDEGRADED\x1b[0m"
	}
	if noColor {
		return "OK"
	}
	return "\x1b[32mOK\x1b[0m"
}

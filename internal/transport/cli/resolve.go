package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"sort"

	"github.com/spf13/cobra"

	"github.com/soundcheck/soundcheck/internal/app"
	"github.com/soundcheck/soundcheck/internal/core/domain"
)

func newResolveCmd(opts *RootOptions, injected *app.App) *cobra.Command {
	return &cobra.Command{
		Use:   "resolve [role]",
		Short: "Output resolved agent/model assignments",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, cancel := resolveContext(cmd.Context(), opts.Timeout)
			defer cancel()

			a, err := resolveApp(opts, injected)
			if err != nil {
				return err
			}

			roleFilter := ""
			if len(args) == 1 {
				roleFilter = args[0]
			}

			res, err := a.Resolve(ctx, roleFilter)
			if err != nil {
				return &ExitError{Code: ExitFailed, Err: err}
			}

			if roleFilter != "" && len(res) == 0 {
				return &ExitError{Code: ExitUsage, Err: fmt.Errorf("role not found: %s", roleFilter)}
			}

			if err := outputResolutions(cmd.OutOrStdout(), opts, res); err != nil {
				return err
			}
			return resolveExitCode(res)
		},
	}
}

func resolveExitCode(res map[domain.Role]domain.DetailedResolution) error {
	for _, r := range res {
		if !r.Healthy || r.SelectedAgentID == "" {
			return &ExitError{Code: ExitDegraded}
		}
	}
	return nil
}

func outputResolutions(w io.Writer, opts *RootOptions, res map[domain.Role]domain.DetailedResolution) error {
	if opts.JSON {
		b, err := json.MarshalIndent(res, "", "  ")
		if err != nil {
			return &ExitError{Code: ExitFailed, Err: err}
		}
		_, err = fmt.Fprintln(w, string(b))
		return err
	}
	if opts.NDJSON {
		return outputResolutionsNDJSON(w, res)
	}
	return renderResolutionsText(w, res, isColorDisabled(opts))
}

func outputResolutionsNDJSON(w io.Writer, res map[domain.Role]domain.DetailedResolution) error {
	enc := json.NewEncoder(w)
	roles := sortedRoles(res)
	for _, role := range roles {
		item := map[string]any{"role": role, "resolution": res[role]}
		if err := enc.Encode(item); err != nil {
			return err
		}
	}
	return nil
}

func renderResolutionsText(w io.Writer, res map[domain.Role]domain.DetailedResolution, noColor bool) error {
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
		if _, err := fmt.Fprintf(w, "%-14s %-10s (%s, %s) [%s]\n", role, r.SelectedAgentID, r.SelectedModel, r.Vendor, status); err != nil {
			return err
		}
		if r.Reasoning != "" {
			if _, err := fmt.Fprintf(w, "  └─ %s\n", r.Reasoning); err != nil {
				return err
			}
		}
	}
	return nil
}

func sortedRoles(res map[domain.Role]domain.DetailedResolution) []domain.Role {
	roles := make([]domain.Role, 0, len(res))
	for r := range res {
		roles = append(roles, r)
	}
	sort.Slice(roles, func(i, j int) bool { return roles[i] < roles[j] })
	return roles
}

func colorizeStatus(status string) string {
	if status == "HEALTHY" {
		return "\x1b[32mHEALTHY\x1b[0m"
	}
	return "\x1b[31mUNHEALTHY\x1b[0m"
}

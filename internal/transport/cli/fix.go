package cli

import (
	"encoding/json"
	"fmt"

	"github.com/soundcheck/soundcheck/internal/app"
	"github.com/spf13/cobra"
)

func newFixCmd(opts *RootOptions, a *app.App) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "fix [adapter]",
		Short: "Remediate and repair broken or unauthenticated agent adapters",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			appInstance, err := resolveApp(opts, a)
			if err != nil {
				return err
			}

			adapterID := args[0]
			ctx, cancel := resolveContext(cmd.Context(), opts.Timeout)
			defer cancel()

			res, err := appInstance.Fix(ctx, adapterID)
			if err != nil {
				return &ExitError{Code: ExitFailed, Err: err}
			}

			out := cmd.OutOrStdout()
			if opts.JSON {
				enc := json.NewEncoder(out)
				enc.SetIndent("", "  ")
				return enc.Encode(res)
			}

			_, _ = fmt.Fprintf(out, "REMEDIATION FOR %s:\n", res.AdapterID)
			_, _ = fmt.Fprintf(out, "  Action:  %s\n", res.Description)
			_, _ = fmt.Fprintf(out, "  Command: %s\n", res.RemediationCmd)
			return nil
		},
	}
	return cmd
}

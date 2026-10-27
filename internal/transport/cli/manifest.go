package cli

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/soundcheck/soundcheck/internal/app"
)

func newManifestCmd(opts *RootOptions, injected *app.App) *cobra.Command {
	return &cobra.Command{
		Use:   "manifest",
		Short: "Output capability manifest JSON",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			ctx, cancel := resolveContext(cmd.Context(), opts.Timeout)
			defer cancel()

			a, err := resolveApp(opts, injected)
			if err != nil {
				return err
			}

			m, err := a.Manifest(ctx)
			if err != nil {
				return &ExitError{Code: ExitFailed, Err: err}
			}

			return outputManifest(cmd, opts, m)
		},
	}
}

func outputManifest(cmd *cobra.Command, opts *RootOptions, m any) error {
	out := cmd.OutOrStdout()
	if opts.NDJSON {
		enc := json.NewEncoder(out)
		return enc.Encode(map[string]any{"type": "manifest", "data": m})
	}
	bytes, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return &ExitError{Code: ExitFailed, Err: err}
	}
	_, err = fmt.Fprintln(out, string(bytes))
	return err
}

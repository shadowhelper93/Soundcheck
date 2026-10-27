package cli

import (
	"github.com/soundcheck/soundcheck/internal/app"
	"github.com/soundcheck/soundcheck/internal/transport/mcp"
	"github.com/spf13/cobra"
)

func newMCPCmd(opts *RootOptions, a *app.App) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "mcp",
		Short: "Start Model Context Protocol (MCP) stdio JSON-RPC server",
		RunE: func(cmd *cobra.Command, _ []string) error {
			appInstance, err := resolveApp(opts, a)
			if err != nil {
				return err
			}
			if err := mcp.ServeStdio(cmd.Context(), appInstance, cmd.InOrStdin(), cmd.OutOrStdout()); err != nil {
				return &ExitError{Code: ExitFailed, Err: err}
			}
			return nil
		},
	}
	return cmd
}

package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/spf13/cobra"

	"github.com/soundcheck/soundcheck/internal/app"
)

type execOptions struct {
	Adapter string
	Role    string
}

func newExecCmd(opts *RootOptions, a *app.App) *cobra.Command {
	var execOpts execOptions
	cmd := &cobra.Command{
		Use:   "exec <prompt>",
		Short: "Execute a prompt headlessly with automatic quirk classification and failover",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			prompt := strings.Join(args, " ")
			return runExec(cmd, opts, a, prompt)
		},
	}
	cmd.Flags().StringVarP(&execOpts.Adapter, "adapter", "a", "", "Target a specific adapter")
	cmd.Flags().StringVarP(&execOpts.Role, "role", "r", "", "Target the adapter assigned to a role")
	return cmd
}

func runExec(cmd *cobra.Command, opts *RootOptions, a *app.App, prompt string) error {
	appInstance, err := resolveApp(opts, a)
	if err != nil {
		return err
	}
	ctx, cancel := resolveContext(cmd.Context(), opts.Timeout)
	defer cancel()

	var target string
	if adapter, _ := cmd.Flags().GetString("adapter"); adapter != "" {
		target = adapter
	} else if role, _ := cmd.Flags().GetString("role"); role != "" {
		target = role
	}

	res, err := appInstance.RunPrompt(ctx, prompt, target)
	if err != nil {
		return &ExitError{Code: ExitUsage, Err: err}
	}

	if opts.JSON {
		return outputExecJSON(cmd.OutOrStdout(), res)
	}
	return outputExecText(cmd.OutOrStdout(), cmd.ErrOrStderr(), res)
}

func outputExecJSON(out io.Writer, res *app.RunPromptResult) error {
	enc := json.NewEncoder(out)
	enc.SetIndent("", "  ")
	return enc.Encode(res)
}

func outputExecText(out io.Writer, errOut io.Writer, res *app.RunPromptResult) error {
	if res.FailoverOccurred {
		for _, att := range res.Attempts {
			if !att.Success {
				_, _ = fmt.Fprintf(errOut, "[soundcheck: %s failed: %s -> falling over]\n", att.AgentID, att.ErrorMsg)
			}
		}
	}
	if !res.Success {
		_, _ = fmt.Fprintln(errOut, res.Output)
		return &ExitError{Code: ExitFailed, Err: fmt.Errorf("all candidate agents failed")}
	}
	_, _ = fmt.Fprintln(out, res.Output)
	return nil
}

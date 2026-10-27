// Package cli implements the Soundcheck command line interface transport.
package cli

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/spf13/cobra"

	"github.com/soundcheck/soundcheck/internal/app"
	"github.com/soundcheck/soundcheck/internal/edge/config"
)

// Standard exit codes.
const (
	ExitOK            = 0
	ExitDegraded      = 1
	ExitFailed        = 2
	ExitUsage         = 3
	ExitConfigInvalid = 4
)

// RootOptions holds persistent global CLI flags.
type RootOptions struct {
	Config  string
	JSON    bool
	NDJSON  bool
	NoColor bool
	Timeout time.Duration
	Verbose bool
}

// ExecutionConfig configures CLI execution streams and dependencies.
type ExecutionConfig struct {
	Args []string
	Out  io.Writer
	Err  io.Writer
	App  *app.App
}

// ExitError captures an exit code along with an underlying error.
type ExitError struct {
	Code int
	Err  error
}

func (e *ExitError) Error() string {
	if e.Err != nil {
		return e.Err.Error()
	}
	return fmt.Sprintf("exit status %d", e.Code)
}

func (e *ExitError) Unwrap() error {
	return e.Err
}

// Execute runs the default root command against os.Args and returns exit code.
func Execute() int {
	return ExecuteCustom(context.Background(), ExecutionConfig{
		Args: os.Args[1:],
		Out:  os.Stdout,
		Err:  os.Stderr,
	})
}

// ExecuteCustom runs the root command with explicit context and configuration.
func ExecuteCustom(ctx context.Context, cfg ExecutionConfig) int {
	opts := &RootOptions{}
	cmd := NewRootCmd(opts, cfg.App)
	cmd.SetArgs(cfg.Args)
	if cfg.Out != nil {
		cmd.SetOut(cfg.Out)
	}
	if cfg.Err != nil {
		cmd.SetErr(cfg.Err)
	}

	err := cmd.ExecuteContext(ctx)
	return extractExitCode(err)
}

func extractExitCode(err error) int {
	if err == nil {
		return ExitOK
	}
	var exitErr *ExitError
	if errors.As(err, &exitErr) {
		return exitErr.Code
	}
	return ExitUsage
}

// NewRootCmd constructs the soundcheck Cobra root command.
func NewRootCmd(opts *RootOptions, a *app.App) *cobra.Command {
	cmd := &cobra.Command{
		Use:           "soundcheck",
		Short:         "Soundcheck CLI - preflight prober and agent resolver",
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	addPersistentFlags(cmd, opts)
	addCommands(cmd, opts, a)
	return cmd
}

func addPersistentFlags(cmd *cobra.Command, opts *RootOptions) {
	f := cmd.PersistentFlags()
	f.StringVarP(&opts.Config, "config", "c", "", "Path to soundcheck configuration file")
	f.BoolVar(&opts.JSON, "json", false, "Output machine-readable JSON")
	f.BoolVar(&opts.NDJSON, "ndjson", false, "Output newline-delimited JSON stream")
	f.BoolVar(&opts.NoColor, "no-color", false, "Disable ANSI color output")
	f.DurationVarP(&opts.Timeout, "timeout", "t", 0, "Execution timeout duration (e.g. 10s)")
	f.BoolVarP(&opts.Verbose, "verbose", "v", false, "Enable verbose output")
}

func addCommands(cmd *cobra.Command, opts *RootOptions, a *app.App) {
	cmd.AddCommand(newDoctorCmd(opts, a))
	cmd.AddCommand(newProbeCmd(opts, a))
	cmd.AddCommand(newResolveCmd(opts, a))
	cmd.AddCommand(newManifestCmd(opts, a))
	cmd.AddCommand(newVersionCmd(opts))
}

func resolveApp(opts *RootOptions, injected *app.App) (*app.App, error) {
	if injected != nil {
		return injected, nil
	}
	var appOpts []app.Option
	if opts.Config != "" {
		cfg, err := config.Load(opts.Config)
		if err != nil {
			return nil, &ExitError{Code: ExitConfigInvalid, Err: fmt.Errorf("config error: %w", err)}
		}
		appOpts = append(appOpts, app.WithConfig(cfg))
	}
	application, err := app.New(appOpts...)
	if err != nil {
		return nil, &ExitError{Code: ExitFailed, Err: err}
	}
	return application, nil
}

func resolveContext(ctx context.Context, timeout time.Duration) (context.Context, context.CancelFunc) {
	if timeout > 0 {
		return context.WithTimeout(ctx, timeout)
	}
	return ctx, func() {}
}

func isColorDisabled(opts *RootOptions) bool {
	return opts.NoColor || os.Getenv("NO_COLOR") != ""
}

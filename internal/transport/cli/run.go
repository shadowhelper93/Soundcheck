package cli

import (
	"io"
	"os"

	"github.com/spf13/cobra"
	"golang.org/x/term"

	"github.com/soundcheck/soundcheck/internal/adapters"
	"github.com/soundcheck/soundcheck/internal/app"
	"github.com/soundcheck/soundcheck/internal/edge/exec"
)

func toWinsize(w, h int) exec.Winsize {
	cols := uint16(80)
	rows := uint16(24)
	if w > 0 && w <= 65535 {
		// #nosec G115
		cols = uint16(w)
	}
	if h > 0 && h <= 65535 {
		// #nosec G115
		rows = uint16(h)
	}
	return exec.Winsize{Cols: cols, Rows: rows}
}

func newRunCmd(opts *RootOptions, _ *app.App) *cobra.Command {
	return &cobra.Command{
		Use:   "run <adapter> [-- <args...>]",
		Short: "Run an adapter interactively in a PTY with raw terminal passthrough",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return executeRun(cmd, opts, args)
		},
	}
}

func executeRun(cmd *cobra.Command, opts *RootOptions, args []string) error {
	ctx, cancel := resolveContext(cmd.Context(), opts.Timeout)
	defer cancel()
	cmd.SetContext(ctx)

	target := resolveRunTarget(args[0])
	state, fd, ws := setupTerminal(cmd.InOrStdin())
	if state != nil {
		defer func() { _ = term.Restore(fd, state) }()
	}

	runner := exec.NewPTYRunner()
	session, err := runner.Start(ctx, exec.Cmd{Binary: target, Args: args[1:]}, ws)
	if err != nil {
		return &ExitError{Code: ExitFailed, Err: err}
	}
	defer func() { _ = session.Close() }()

	done := make(chan struct{})
	defer close(done)
	startResizeWatcher(done, fd, session)
	go forwardStdin(done, cmd.InOrStdin(), session)

	return finishRun(session, cmd.OutOrStdout())
}

func finishRun(session exec.PTYSession, out io.Writer) error {
	exitCode := streamEvents(session.Events(), out)
	waitCode, waitErr := session.Wait()
	if waitErr != nil && exitCode == 0 {
		exitCode = waitCode
	}
	if exitCode != 0 {
		return &ExitError{Code: exitCode, Err: waitErr}
	}
	return nil
}

func resolveRunTarget(id string) string {
	if def, ok := adapters.Get(id); ok && def.Binary != "" {
		return def.Binary
	}
	return id
}

func setupTerminal(r io.Reader) (*term.State, int, exec.Winsize) {
	ws := exec.Winsize{Cols: 80, Rows: 24}
	f, ok := r.(*os.File)
	if !ok || !term.IsTerminal(int(f.Fd())) {
		return nil, -1, ws
	}
	fd := int(f.Fd())
	if w, h, err := term.GetSize(fd); err == nil && w > 0 && h > 0 {
		ws = toWinsize(w, h)
	}
	state, err := term.MakeRaw(fd)
	if err != nil {
		return nil, -1, ws
	}
	return state, fd, ws
}

func startResizeWatcher(done <-chan struct{}, fd int, session exec.PTYSession) {
	if fd < 0 {
		return
	}
	sigCh := make(chan os.Signal, 1)
	notifySIGWINCH(sigCh)
	go func() {
		defer stopSIGWINCH(sigCh)
		watchResize(done, sigCh, fd, session)
	}()
}

func watchResize(done <-chan struct{}, sigCh <-chan os.Signal, fd int, session exec.PTYSession) {
	for {
		select {
		case <-done:
			return
		case _, ok := <-sigCh:
			if !ok {
				return
			}
			if w, h, err := term.GetSize(fd); err == nil && w > 0 && h > 0 {
				ws := toWinsize(w, h)
				_ = session.Resize(ws.Cols, ws.Rows)
			}
		}
	}
}

func forwardStdin(done <-chan struct{}, in io.Reader, session exec.PTYSession) {
	buf := make([]byte, 1024)
	for {
		select {
		case <-done:
			return
		default:
		}
		n, err := in.Read(buf)
		if n > 0 {
			if _, werr := session.Write(buf[:n]); werr != nil {
				return
			}
		}
		if err != nil {
			return
		}
	}
}

func streamEvents(events <-chan exec.Event, out io.Writer) int {
	exitCode := 0
	for ev := range events {
		switch ev.Type {
		case exec.EventOutput, exec.EventStdout:
			_, _ = out.Write(ev.Data)
		case exec.EventExit:
			exitCode = ev.ExitCode
		}
	}
	return exitCode
}

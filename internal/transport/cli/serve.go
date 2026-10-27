package cli

import (
	"fmt"
	"io/fs"

	"github.com/soundcheck/soundcheck/internal/app"
	goserver "github.com/soundcheck/soundcheck/internal/transport/http"
	"github.com/spf13/cobra"
)

var defaultAssets fs.FS

// SetDefaultAssets sets the embedded static UI assets for the serve command.
func SetDefaultAssets(assets fs.FS) {
	defaultAssets = assets
}

type serveFlags struct {
	host  string
	port  int
	token string
}

func newServeCmd(opts *RootOptions, a *app.App) *cobra.Command {
	flags := &serveFlags{}
	cmd := &cobra.Command{
		Use:   "serve",
		Short: "Start HTTP server, REST API, and web UI",
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runServe(cmd, opts, a, flags)
		},
	}
	cmd.Flags().StringVar(&flags.host, "host", "127.0.0.1", "host to bind to")
	cmd.Flags().IntVarP(&flags.port, "port", "p", 3000, "port to listen on")
	cmd.Flags().StringVar(&flags.token, "token", "", "optional Bearer auth token")
	return cmd
}

func runServe(cmd *cobra.Command, opts *RootOptions, a *app.App, flags *serveFlags) error {
	appInstance, err := resolveApp(opts, a)
	if err != nil {
		return err
	}
	addr := fmt.Sprintf("%s:%d", flags.host, flags.port)
	srvOpts := []goserver.ServerOption{goserver.WithAddr(addr)}
	if flags.token != "" {
		srvOpts = append(srvOpts, goserver.WithAuthToken(flags.token))
	}
	if defaultAssets != nil {
		srvOpts = append(srvOpts, goserver.WithAssets(defaultAssets))
	}
	srv := goserver.NewServer(appInstance, srvOpts...)
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Soundcheck server listening at http://%s\n", addr)
	if err := srv.Start(cmd.Context()); err != nil {
		return &ExitError{Code: ExitFailed, Err: err}
	}
	return nil
}

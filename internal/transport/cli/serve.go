package cli

import (
	"fmt"

	"github.com/soundcheck/soundcheck/internal/app"
	goserver "github.com/soundcheck/soundcheck/internal/transport/http"
	"github.com/spf13/cobra"
)

func newServeCmd(opts *RootOptions, a *app.App) *cobra.Command {
	var (
		host  string
		port  int
		token string
	)
	cmd := &cobra.Command{
		Use:   "serve",
		Short: "Start HTTP server, REST API, and web UI",
		RunE: func(cmd *cobra.Command, _ []string) error {
			appInstance, err := resolveApp(opts, a)
			if err != nil {
				return err
			}
			addr := fmt.Sprintf("%s:%d", host, port)
			var srvOpts []goserver.ServerOption
			srvOpts = append(srvOpts, goserver.WithAddr(addr))
			if token != "" {
				srvOpts = append(srvOpts, goserver.WithAuthToken(token))
			}
			srv := goserver.NewServer(appInstance, srvOpts...)
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Soundcheck server listening at http://%s\n", addr)
			if err := srv.Start(cmd.Context()); err != nil {
				return &ExitError{Code: ExitFailed, Err: err}
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&host, "host", "127.0.0.1", "host to bind to")
	cmd.Flags().IntVarP(&port, "port", "p", 3000, "port to listen on")
	cmd.Flags().StringVar(&token, "token", "", "optional Bearer auth token")
	return cmd
}

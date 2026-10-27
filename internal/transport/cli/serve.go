package cli

import (
	"fmt"
	"net/http"
	"time"

	"github.com/soundcheck/soundcheck/internal/app"
	goserver "github.com/soundcheck/soundcheck/internal/transport/http"
	"github.com/spf13/cobra"
)

func newServeCmd(opts *RootOptions, a *app.App) *cobra.Command {
	var port int
	cmd := &cobra.Command{
		Use:   "serve",
		Short: "Start HTTP server and web UI dashboard",
		RunE: func(cmd *cobra.Command, _ []string) error {
			appInstance, err := resolveApp(opts, a)
			if err != nil {
				return err
			}

			srv := goserver.NewServer(appInstance)
			addr := fmt.Sprintf(":%d", port)
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Soundcheck server running at http://localhost%s\n", addr)

			httpServer := &http.Server{
				Addr:              addr,
				Handler:           srv,
				ReadHeaderTimeout: 5 * time.Second,
				ReadTimeout:       15 * time.Second,
				WriteTimeout:      30 * time.Second,
				IdleTimeout:       60 * time.Second,
			}

			if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
				return &ExitError{Code: ExitFailed, Err: err}
			}
			return nil
		},
	}
	cmd.Flags().IntVarP(&port, "port", "p", 8080, "port to listen on")
	return cmd
}

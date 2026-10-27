package cli

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"
)

// Version string for the Soundcheck CLI.
var Version = "v2.0.0"

func newVersionCmd(opts *RootOptions) *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print the Soundcheck CLI version",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			out := cmd.OutOrStdout()
			if opts.JSON || opts.NDJSON {
				data := map[string]string{"version": Version}
				enc := json.NewEncoder(out)
				return enc.Encode(data)
			}
			_, err := fmt.Fprintf(out, "soundcheck %s\n", Version)
			return err
		},
	}
}

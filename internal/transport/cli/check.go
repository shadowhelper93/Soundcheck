package cli

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/soundcheck/soundcheck/internal/app"
	"github.com/soundcheck/soundcheck/internal/core/domain"
	"github.com/spf13/cobra"
)

func newCheckCmd(opts *RootOptions, a *app.App) *cobra.Command {
	var modelName string
	cmd := &cobra.Command{
		Use:   "check [provider]",
		Short: "Inspect a provider and automatically discover verified alternative fallbacks",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runCheck(cmd, opts, a, args[0], modelName)
		},
	}
	cmd.Flags().StringVarP(&modelName, "model", "m", "claude-sonnet-5-5", "model to check alternative providers for")
	return cmd
}

func runCheck(cmd *cobra.Command, opts *RootOptions, a *app.App, providerID, modelName string) error {
	appInstance, err := resolveApp(opts, a)
	if err != nil {
		return err
	}
	ctx, cancel := resolveContext(cmd.Context(), opts.Timeout)
	defer cancel()

	adapter, err := appInstance.ProbeOne(ctx, providerID)
	if err != nil {
		return &ExitError{Code: ExitFailed, Err: err}
	}
	out := cmd.OutOrStdout()
	printProviderHeader(out, adapter)

	if !adapter.Installed {
		return nil
	}

	if modelName == "" {
		modelName = "claude-sonnet-5-5"
	}
	alts, err := appInstance.FindAlternativesForModel(ctx, providerID, modelName)
	if err != nil || len(alts) == 0 {
		_, _ = fmt.Fprintf(out, "  No other verified providers found hosting model %q.\n", modelName)
		return nil
	}

	promptAddAlternatives(out, opts.Config, modelName, alts)
	return nil
}

func printProviderHeader(out io.Writer, adapter domain.Adapter) {
	_, _ = fmt.Fprintf(out, "CHECKING PROVIDER: %s (%s)\n", adapter.Name, adapter.ID)
	isAuth := false
	if p, ok := adapter.Probes[domain.TierL1]; ok && p.ErrorClass == domain.ErrorClassOK {
		isAuth = true
	}
	if !adapter.Installed {
		_, _ = fmt.Fprintf(out, "  Status: Not Installed in $PATH\n")
		return
	}
	if !isAuth {
		_, _ = fmt.Fprintf(out, "  Status: Unauthenticated or Quota Exhausted\n")
		return
	}
	_, _ = fmt.Fprintf(out, "  Status: Healthy & Authenticated\n")
}

func promptAddAlternatives(out io.Writer, cfgPath, modelName string, alts []app.VerifiedAlternative) {
	_, _ = fmt.Fprintf(out, "\nDISCOVERED VERIFIED WORKING ALTERNATIVES FOR '%s':\n", modelName)
	for _, alt := range alts {
		_, _ = fmt.Fprintf(out, "  - %s (model: %s) -> [VERIFIED WORKING]\n", alt.ProviderID, alt.ModelID)
	}

	_, _ = fmt.Fprintf(out, "\nThese verified providers are ready on this machine. Add to soundcheck.yaml config? [y/N]: ")
	reader := bufio.NewReader(os.Stdin)
	ans, _ := reader.ReadString('\n')
	ans = strings.TrimSpace(strings.ToLower(ans))

	if ans == "y" || ans == "yes" {
		for _, alt := range alts {
			_ = app.SaveFallbackToConfig(cfgPath, modelName, alt.ProviderID)
		}
		_, _ = fmt.Fprintf(out, "Saved verified fallbacks to soundcheck.yaml\n")
	} else {
		_, _ = fmt.Fprintf(out, "Skipped updating config.\n")
	}
}

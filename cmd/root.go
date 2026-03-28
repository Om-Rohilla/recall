package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var (
	Version   = "dev"
	BuildDate = "unknown"
	debug     bool
)

var rootCmd = &cobra.Command{
	Use:   "recall",
	Short: "Your terminal has perfect memory",
	Long: `Recall — Command Intelligence Engine

100% offline CLI tool that eliminates Command Amnesia.
Search commands by intent, not exact syntax.
Nothing ever leaves your machine.`,
	SilenceUsage:  true,
	SilenceErrors: true,
	Args:          cobra.ArbitraryArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) == 0 {
			return cmd.Help()
		}
		return runSearch(args)
	},
}

func Execute() error {
	return rootCmd.Execute()
}

func init() {
	rootCmd.PersistentFlags().BoolVar(&debug, "debug", false, "enable debug output")

	if os.Getenv("NO_COLOR") != "" {
		// Color disabling is handled in the UI layer
	}

	rootCmd.AddCommand(versionCmd)
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Show version info",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("recall %s\n  built: %s\n  go:    %s\n", Version, BuildDate, "go1.24.1")
	},
}

// runSearch is called when the first arg isn't a known subcommand.
// Delegates to the search command logic.
func runSearch(args []string) error {
	// Will be wired to search logic in cmd/search.go
	return fmt.Errorf("search not yet implemented — run 'recall init' first")
}

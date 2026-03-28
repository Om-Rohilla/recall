package cmd

import (
	"fmt"
	"os"
	"runtime"

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
		fmt.Printf("recall %s\n  built:   %s\n  go:      %s\n  os:      %s/%s\n",
			Version, BuildDate, runtime.Version(), runtime.GOOS, runtime.GOARCH)
	},
}

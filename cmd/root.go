package cmd

import (
	"fmt"
	"os"
	"runtime"

	"github.com/Om-Rohilla/recall/pkg/logging"
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
	logging.Init()
	return rootCmd.Execute()
}

func init() {
	rootCmd.PersistentFlags().BoolVar(&debug, "debug", false, "enable debug output")

	if os.Getenv("NO_COLOR") != "" {
		// Color disabling is handled in the UI layer
	}

	rootCmd.AddCommand(versionCmd)
	rootCmd.AddCommand(completionCmd)
}

var completionCmd = &cobra.Command{
	Use:   "completion [bash|zsh|fish|powershell]",
	Short: "Generate shell completion scripts",
	Long: `Generate shell completion scripts for Recall.

To load completions:

Bash:   $ source <(recall completion bash)
Zsh:    $ recall completion zsh > "${fpath[1]}/_recall"
Fish:   $ recall completion fish | source

To load permanently, add the above to your shell config.`,
	DisableFlagsInUseLine: true,
	ValidArgs:             []string{"bash", "zsh", "fish", "powershell"},
	Args:                  cobra.MatchAll(cobra.ExactArgs(1), cobra.OnlyValidArgs),
	RunE: func(cmd *cobra.Command, args []string) error {
		switch args[0] {
		case "bash":
			return rootCmd.GenBashCompletion(os.Stdout)
		case "zsh":
			return rootCmd.GenZshCompletion(os.Stdout)
		case "fish":
			return rootCmd.GenFishCompletion(os.Stdout, true)
		case "powershell":
			return rootCmd.GenPowerShellCompletionWithDesc(os.Stdout)
		default:
			return fmt.Errorf("unsupported shell: %s", args[0])
		}
	},
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Show version info",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("recall %s\n  built:   %s\n  go:      %s\n  os:      %s/%s\n",
			Version, BuildDate, runtime.Version(), runtime.GOOS, runtime.GOARCH)
	},
}

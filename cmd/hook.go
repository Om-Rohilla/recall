package cmd

import (
	"fmt"

	"github.com/Om-Rohilla/recall/pkg/shell"
	"github.com/spf13/cobra"
)

var hookCmd = &cobra.Command{
	Use:   "hook [shell]",
	Short: "Output shell hook script for eval",
	Long: `Output the Recall shell hook script for the specified shell.

Usage in your shell config:
  eval "$(recall hook zsh)"
  eval "$(recall hook bash)"`,
	Args: cobra.ExactArgs(1),
	RunE: runHook,
}

func init() {
	rootCmd.AddCommand(hookCmd)
}

func runHook(cmd *cobra.Command, args []string) error {
	shellName := args[0]
	info, err := shell.ForShell(shellName)
	if err != nil {
		return err
	}

	script, err := shell.HookScript(info.Shell)
	if err != nil {
		return err
	}

	fmt.Print(script)
	return nil
}

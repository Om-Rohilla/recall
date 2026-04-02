package cmd

import (
	"fmt"

	"github.com/Om-Rohilla/recall/internal/ui"
	"github.com/Om-Rohilla/recall/internal/vault"
	"github.com/Om-Rohilla/recall/pkg/config"
	"github.com/spf13/cobra"
)

var wrappedCmd = &cobra.Command{
	Use:   "wrapped",
	Short: "View your Terminal Wrapped (Weekly Summary)",
	RunE:  runWrapped,
}

func init() {
	rootCmd.AddCommand(wrappedCmd)
}

func runWrapped(cmd *cobra.Command, args []string) error {
	cfg := config.Get()

	store, err := vault.NewStore(cfg.Vault.Path)
	if err != nil {
		return fmt.Errorf("opening vault: %w", err)
	}
	defer store.Close()

	stats, err := store.GetWrappedStats()
	if err != nil {
		return fmt.Errorf("generating wrapped stats: %w", err)
	}

	ui.ShowWrapped(stats)
	return nil
}

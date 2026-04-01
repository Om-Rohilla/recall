package cmd

import (
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"github.com/Om-Rohilla/recall/internal/ui"
	"github.com/Om-Rohilla/recall/internal/vault"
	"github.com/Om-Rohilla/recall/pkg/config"
)

var contextCmd = &cobra.Command{
	Use:   "context",
	Short: "Manage execution contexts",
	Long: `Manage the execution context metadata stored in your vault.

Contexts include working directories, git repositories, exit codes,
and command durations associated with your command history.

Subcommands:
  stats   Show context statistics
  prune   Delete orphaned contexts`,
}

var ctxStatsCmd = &cobra.Command{
	Use:   "stats",
	Short: "Show context statistics",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg := config.Get()
		store, err := vault.NewStore(cfg.Vault.Path)
		if err != nil {
			return fmt.Errorf("opening vault: %w", err)
		}
		defer store.Close()

		stats, err := store.GetContextStats()
		if err != nil {
			return fmt.Errorf("getting context stats: %w", err)
		}

		fmt.Println(ui.TitleStyle.Render("  Context Statistics"))
		fmt.Println()
		fmt.Printf("  Total Executions: %d\n", stats.TotalContexts)
		fmt.Printf("  Unique WorkDirs:  %d\n", stats.UniqueCwds)
		fmt.Printf("  Unique Git Repos: %d\n", stats.UniqueGitRepos)
		fmt.Printf("  Unique Sessions:  %d\n", stats.UniqueSessions)
		
		if !stats.EarliestContext.IsZero() {
			fmt.Printf("  Tracking Since:   %s (%d days ago)\n",
				stats.EarliestContext.Format("2006-01-02"),
				int(time.Since(stats.EarliestContext).Hours()/24))
		}
		return nil
	},
}

var ctxPruneCmd = &cobra.Command{
	Use:   "prune",
	Short: "Delete orphaned contexts",
	Long:  "Deletes contexts that reference command IDs that no longer exist (e.g., after deletion or pruning).",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg := config.Get()
		store, err := vault.NewStore(cfg.Vault.Path)
		if err != nil {
			return fmt.Errorf("opening vault: %w", err)
		}
		defer store.Close()

		fmt.Println(ui.WarningStyle.Render("  Pruning orphaned contexts..."))
		
		pruned, err := store.PruneOrphanedContexts()
		if err != nil {
			return fmt.Errorf("pruning contexts: %w", err)
		}

		if pruned > 0 {
			fmt.Printf("  %s Deleted %d orphaned contexts.\n", ui.SuccessStyle.Render("✓"), pruned)
		} else {
			fmt.Printf("  %s No orphaned contexts found.\n", ui.HintStyle.Render("·"))
		}
		return nil
	},
}

func init() {
	contextCmd.AddCommand(ctxStatsCmd)
	contextCmd.AddCommand(ctxPruneCmd)
	rootCmd.AddCommand(contextCmd)
}

package cmd

import (
	"fmt"

	"github.com/dustin/go-humanize"
	"github.com/spf13/cobra"

	"github.com/Om-Rohilla/recall/internal/ui"
	"github.com/Om-Rohilla/recall/internal/vault"
	"github.com/Om-Rohilla/recall/pkg/config"
	"github.com/Om-Rohilla/recall/pkg/logging"
)

var (
	maintVacuum       bool
	maintRebuildIndex bool
	maintPruneBefore  int
	maintRotateLogs   bool
)

var maintenanceCmd = &cobra.Command{
	Use:   "maintenance",
	Short: "Vault maintenance and optimization",
	Long: `Run maintenance tasks on your Recall vault.

By default, runs vacuum + index rebuild and reports vault size.
Use flags to select specific operations.

Examples:
  recall maintenance
  recall maintenance --vacuum
  recall maintenance --rebuild-index
  recall maintenance --prune-before 365
  recall maintenance --rotate-logs`,
	RunE: runMaintenance,
}

func init() {
	maintenanceCmd.Flags().BoolVar(&maintVacuum, "vacuum", false, "run SQLite VACUUM to reclaim space")
	maintenanceCmd.Flags().BoolVar(&maintRebuildIndex, "rebuild-index", false, "rebuild FTS5 search indexes")
	maintenanceCmd.Flags().IntVar(&maintPruneBefore, "prune-before", 0, "prune commands older than N days")
	maintenanceCmd.Flags().BoolVar(&maintRotateLogs, "rotate-logs", false, "force log file rotation")
	rootCmd.AddCommand(maintenanceCmd)
}

func runMaintenance(cmd *cobra.Command, args []string) error {
	cfg := config.Get()

	// If no specific flags given, run the default set (vacuum + rebuild)
	noFlags := !maintVacuum && !maintRebuildIndex && maintPruneBefore == 0 && !maintRotateLogs
	if noFlags {
		maintVacuum = true
		maintRebuildIndex = true
	}

	store, err := vault.NewStore(cfg.Vault.Path)
	if err != nil {
		return fmt.Errorf("opening vault: %w", err)
	}
	defer store.Close()

	fmt.Println(ui.TitleStyle.Render("  🔧 Recall Maintenance"))
	fmt.Println()

	// Report vault size before
	sizeBefore, _ := store.VaultFileSize()
	stats, _ := store.GetStats()
	if stats != nil {
		fmt.Printf("  📦 Vault: %s (%d commands, %d unique)\n",
			humanize.Bytes(uint64(sizeBefore)), stats.TotalCommands, stats.UniqueCommands)
	}
	fmt.Println()

	// Prune old commands
	if maintPruneBefore > 0 {
		pruned, err := store.PruneOldCommands(maintPruneBefore)
		if err != nil {
			fmt.Printf("  %s Prune failed: %v\n", ui.ErrorStyle.Render("✗"), err)
		} else if pruned > 0 {
			fmt.Printf("  %s Pruned %d commands older than %d days\n",
				ui.SuccessStyle.Render("✓"), pruned, maintPruneBefore)
		} else {
			fmt.Printf("  %s No commands older than %d days\n",
				ui.HintStyle.Render("·"), maintPruneBefore)
		}
	}

	// Rebuild FTS indexes
	if maintRebuildIndex {
		if err := store.RebuildFTSIndex(); err != nil {
			fmt.Printf("  %s FTS index rebuild failed: %v\n", ui.ErrorStyle.Render("✗"), err)
		} else {
			fmt.Printf("  %s Commands FTS index rebuilt\n", ui.SuccessStyle.Render("✓"))
		}
		if err := store.RebuildKnowledgeFTSIndex(); err != nil {
			fmt.Printf("  %s Knowledge FTS index rebuild failed: %v\n", ui.ErrorStyle.Render("✗"), err)
		} else {
			fmt.Printf("  %s Knowledge FTS index rebuilt\n", ui.SuccessStyle.Render("✓"))
		}
	}

	// Vacuum
	if maintVacuum {
		if err := store.Vacuum(); err != nil {
			fmt.Printf("  %s Vacuum failed: %v\n", ui.ErrorStyle.Render("✗"), err)
		} else {
			fmt.Printf("  %s Database vacuumed\n", ui.SuccessStyle.Render("✓"))
		}
	}

	// Log rotation
	if maintRotateLogs {
		if err := logging.RotateLogs(); err != nil {
			fmt.Printf("  %s Log rotation failed: %v\n", ui.ErrorStyle.Render("✗"), err)
		} else {
			fmt.Printf("  %s Log files rotated\n", ui.SuccessStyle.Render("✓"))
		}
	}

	// Report vault size after
	sizeAfter, _ := store.VaultFileSize()
	fmt.Println()
	if sizeBefore > 0 && sizeAfter < sizeBefore {
		saved := sizeBefore - sizeAfter
		fmt.Printf("  💾 Vault size: %s → %s (saved %s)\n",
			humanize.Bytes(uint64(sizeBefore)),
			humanize.Bytes(uint64(sizeAfter)),
			humanize.Bytes(uint64(saved)))
	} else {
		fmt.Printf("  💾 Vault size: %s\n", humanize.Bytes(uint64(sizeAfter)))
	}

	fmt.Println()
	fmt.Println(ui.SuccessStyle.Render("  Maintenance complete ✓"))

	return nil
}

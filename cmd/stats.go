package cmd

import (
	"encoding/json"
	"fmt"
	"math"
	"os"

	"github.com/spf13/cobra"

	"github.com/Om-Rohilla/recall/internal/ui"
	"github.com/Om-Rohilla/recall/internal/vault"
	"github.com/Om-Rohilla/recall/pkg/config"
)

var (
	statsPeriod int
	statsJSON   bool
	statsAll    bool
)

var statsCmd = &cobra.Command{
	Use:   "stats",
	Short: "Show usage statistics",
	Long: `Display vault statistics including top commands, categories,
and rare but valuable commands.

Examples:
  recall stats
  recall stats --period 30
  recall stats --json
  recall stats --all`,
	RunE: runStats,
}

func init() {
	statsCmd.Flags().IntVar(&statsPeriod, "period", 7, "stats for last N days")
	statsCmd.Flags().BoolVar(&statsJSON, "json", false, "output as JSON")
	statsCmd.Flags().BoolVar(&statsAll, "all", false, "show all-time stats")
	rootCmd.AddCommand(statsCmd)
}

type StatsOutput struct {
	TotalCommands  int                  `json:"total_commands"`
	UniqueCommands int                  `json:"unique_commands"`
	PeriodDays     int                  `json:"period_days"`
	TopCommands    []vault.Command      `json:"top_commands"`
	Categories     []vault.CategoryCount `json:"categories"`
	RareCommands   []vault.Command      `json:"rare_commands"`
}

func runStats(cmd *cobra.Command, args []string) error {
	cfg := config.Get()

	store, err := vault.NewStore(cfg.Vault.Path)
	if err != nil {
		return fmt.Errorf("opening vault: %w", err)
	}
	defer store.Close()

	stats, err := store.GetStats()
	if err != nil {
		return fmt.Errorf("getting stats: %w", err)
	}

	period := statsPeriod
	if statsAll {
		period = 36500
	}

	topCmds, err := store.GetTopCommands(period, 10)
	if err != nil {
		return fmt.Errorf("getting top commands: %w", err)
	}

	categories, err := store.GetCategories()
	if err != nil {
		return fmt.Errorf("getting categories: %w", err)
	}

	rareCmds, err := store.GetRareCommands(3, 5)
	if err != nil {
		return fmt.Errorf("getting rare commands: %w", err)
	}

	if statsJSON {
		output := StatsOutput{
			TotalCommands:  stats.TotalCommands,
			UniqueCommands: stats.UniqueCommands,
			PeriodDays:     period,
			TopCommands:    topCmds,
			Categories:     categories,
			RareCommands:   rareCmds,
		}
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(output)
	}

	firstSeen, lastSeen, _ := store.GetVaultPeriod()
	captureDays := 0
	if !firstSeen.IsZero() && !lastSeen.IsZero() {
		captureDays = int(math.Ceil(lastSeen.Sub(firstSeen).Hours() / 24))
	}
	if captureDays < 1 && stats.UniqueCommands > 0 {
		captureDays = 1
	}

	uiTopCmds := make([]ui.StatsCommand, len(topCmds))
	for i, c := range topCmds {
		uiTopCmds[i] = ui.StatsCommand{Raw: c.Raw, Frequency: c.Frequency}
	}
	uiCats := make([]ui.StatsCategory, len(categories))
	for i, c := range categories {
		uiCats[i] = ui.StatsCategory{Category: c.Category, Count: c.Count, TotalFrequency: c.TotalFrequency}
	}
	uiRare := make([]ui.StatsCommand, len(rareCmds))
	for i, c := range rareCmds {
		uiRare[i] = ui.StatsCommand{Raw: c.Raw, Frequency: c.Frequency}
	}

	fmt.Println(ui.RenderStats(stats.TotalCommands, stats.UniqueCommands, captureDays, period, uiTopCmds, uiCats, uiRare))
	return nil
}

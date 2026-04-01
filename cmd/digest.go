package cmd

import (
	"fmt"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/Om-Rohilla/recall/internal/ui"
	"github.com/Om-Rohilla/recall/internal/vault"
	"github.com/Om-Rohilla/recall/pkg/config"
)

var digestPeriod int

var digestCmd = &cobra.Command{
	Use:   "digest",
	Short: "Show your weekly command digest",
	Long: `Display a summary of your command activity for the past week.

Shows commands learned, most-used commands, new categories discovered,
your capture streak, and alias suggestions.

Examples:
  recall digest
  recall digest --period 14`,
	RunE: runDigest,
}

func init() {
	digestCmd.Flags().IntVar(&digestPeriod, "period", 7, "digest period in days")
	rootCmd.AddCommand(digestCmd)
}

func runDigest(cmd *cobra.Command, args []string) error {
	cfg := config.Get()

	store, err := vault.NewStore(cfg.Vault.Path)
	if err != nil {
		return fmt.Errorf("opening vault: %w", err)
	}
	defer store.Close()

	since := time.Now().UTC().AddDate(0, 0, -digestPeriod)

	// New commands learned this period
	newCmds, err := store.GetNewCommandsSince(since, 50)
	if err != nil {
		return fmt.Errorf("getting new commands: %w", err)
	}

	// Most-used commands this period
	topCmds, err := store.GetTopCommands(digestPeriod, 5)
	if err != nil {
		return fmt.Errorf("getting top commands: %w", err)
	}

	// New categories discovered
	newCats, err := store.GetCategoriesSince(since)
	if err != nil {
		newCats = nil // non-fatal
	}

	// Streak info
	streak, _ := store.GetCurrentStreak()

	// Stats
	stats, _ := store.GetStats()
	totalCmds := 0
	uniqueCmds := 0
	if stats != nil {
		totalCmds = stats.TotalCommands
		uniqueCmds = stats.UniqueCommands
	}

	// Alias suggestions (reuse suggest logic)
	minFreq := cfg.Alias.MinFrequency
	highFreqCmds, _ := store.GetHighFrequencyCommands(minFreq)
	suggestions := generateAliases(highFreqCmds)
	var uiAliases []ui.AliasSuggestion
	for _, s := range suggestions {
		if len(uiAliases) >= 5 {
			break
		}
		// Only show aliases that actually save typing
		if len(s.Alias) < len(s.Command) && len(strings.TrimSpace(s.Alias)) > 0 {
			uiAliases = append(uiAliases, ui.AliasSuggestion{
				Command:   s.Command,
				Alias:     s.Alias,
				Frequency: s.Frequency,
			})
		}
	}

	digestData := ui.DigestData{
		NewCommands:    newCmds,
		TopCommands:    topCmds,
		NewCategories:  newCats,
		Streak:         &streak,
		Aliases:        uiAliases,
		TotalCommands:  totalCmds,
		UniqueCommands: uniqueCmds,
		PeriodDays:     digestPeriod,
	}

	fmt.Println(ui.RenderDigest(digestData))
	return nil
}

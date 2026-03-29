package cmd

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"strings"

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
		topCmds = nil
	}

	categories, err := store.GetCategories()
	if err != nil {
		categories = nil
	}

	rareCmds, err := store.GetRareCommands(3, 5)
	if err != nil {
		rareCmds = nil
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

	fmt.Println(renderStats(stats, captureDays, period, topCmds, categories, rareCmds))
	return nil
}

func renderStats(stats *vault.VaultStats, captureDays, period int, topCmds []vault.Command, categories []vault.CategoryCount, rareCmds []vault.Command) string {
	var lines []string

	// Title
	lines = append(lines, ui.TitleStyle.Render("Recall Stats"))
	lines = append(lines, "")

	// Overview
	lines = append(lines, ui.MetadataStyle.Render(fmt.Sprintf(
		"  Vault: %s commands captured | %s unique patterns",
		ui.CommandStyle.Render(fmt.Sprintf("%d", stats.TotalCommands)),
		ui.CommandStyle.Render(fmt.Sprintf("%d", stats.UniqueCommands)),
	)))
	if captureDays > 0 {
		lines = append(lines, ui.MetadataStyle.Render(fmt.Sprintf(
			"  Capture period: %s",
			ui.HintStyle.Render(fmt.Sprintf("%d days", captureDays)),
		)))
	}

	// Top Commands
	if len(topCmds) > 0 {
		lines = append(lines, "")
		periodLabel := fmt.Sprintf("this week")
		if period != 7 {
			periodLabel = fmt.Sprintf("last %d days", period)
		}
		lines = append(lines, ui.StatsHeaderStyle.Render(fmt.Sprintf("  Top Commands (%s)", periodLabel)))
		lines = append(lines, "")

		maxFreq := topCmds[0].Frequency
		for i, cmd := range topCmds {
			raw := cmd.Raw
			if len(raw) > 50 {
				raw = raw[:47] + "..."
			}

			barLen := 1
			if maxFreq > 0 {
				barLen = int(float64(cmd.Frequency) / float64(maxFreq) * 15)
				if barLen < 1 {
					barLen = 1
				}
			}
			bar := ui.FrequencyBarStyle.Render(strings.Repeat("█", barLen))

			freqStr := ui.AccentStyle.Render(fmt.Sprintf("(%d)", cmd.Frequency))
			lines = append(lines, fmt.Sprintf("  %s %s %s %s",
				ui.DimStyle.Render(fmt.Sprintf("%2d.", i+1)),
				ui.NormalItemStyle.Render(fmt.Sprintf("%-52s", raw)),
				freqStr,
				bar,
			))
		}
	}

	// Categories
	if len(categories) > 0 {
		lines = append(lines, "")
		lines = append(lines, ui.StatsHeaderStyle.Render("  Top Categories"))
		lines = append(lines, "")

		totalFreq := 0
		for _, c := range categories {
			totalFreq += c.TotalFrequency
		}
		if totalFreq == 0 {
			totalFreq = 1
		}

		maxCats := 8
		if maxCats > len(categories) {
			maxCats = len(categories)
		}

		var catParts []string
		for _, c := range categories[:maxCats] {
			pct := float64(c.TotalFrequency) / float64(totalFreq) * 100
			catParts = append(catParts, ui.CategoryStyle.Render(fmt.Sprintf("%s: %.0f%%", c.Category, pct)))
		}
		lines = append(lines, "  "+strings.Join(catParts, "  "))
	}

	// Rare but valuable
	if len(rareCmds) > 0 {
		lines = append(lines, "")
		lines = append(lines, ui.StatsHeaderStyle.Render("  Rare but Valuable"))
		lines = append(lines, "")

		for i, cmd := range rareCmds {
			raw := cmd.Raw
			if len(raw) > 55 {
				raw = raw[:52] + "..."
			}
			freqLabel := "ever"
			if cmd.Frequency > 1 {
				freqLabel = fmt.Sprintf("%dx, ever", cmd.Frequency)
			}
			lines = append(lines, fmt.Sprintf("  %s %s  %s",
				ui.DimStyle.Render(fmt.Sprintf("%2d.", i+1)),
				ui.NormalItemStyle.Render(raw),
				ui.DimStyle.Render(fmt.Sprintf("(used %s)", freqLabel)),
			))
		}
	}

	content := strings.Join(lines, "\n")
	return ui.BorderStyle.Render(content)
}

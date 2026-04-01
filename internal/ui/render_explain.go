package ui

import (
	"fmt"
	"strings"

	"github.com/Om-Rohilla/recall/internal/explain"
	"github.com/Om-Rohilla/recall/internal/vault"
)

func RenderShortExplain(result explain.ExplainResult) string {
	parts := []string{result.Binary}
	for _, c := range result.Components {
		if c.Type == "subcommand" {
			parts = append(parts, c.Token)
		}
	}
	summary := strings.Join(parts, " ")
	if result.Summary != "" {
		summary = result.Summary
	}

	dangerIcon := ""
	switch result.DangerLevel {
	case explain.Destructive:
		dangerIcon = ErrorStyle.Render(" [DESTRUCTIVE]")
	case explain.Caution:
		dangerIcon = WarningStyle.Render(" [CAUTION]")
	}

	return CommandStyle.Render(result.Raw) + " — " + MetadataStyle.Render(summary) + dangerIcon
}

func RenderFullExplain(result explain.ExplainResult, showWarnings bool) string {
	var lines []string

	titleLabel := "Command Breakdown"
	switch result.DangerLevel {
	case explain.Destructive:
		titleLabel = ErrorStyle.Render("Command Breakdown [DESTRUCTIVE]")
	case explain.Caution:
		titleLabel = WarningStyle.Render("Command Breakdown [CAUTION]")
	default:
		titleLabel = TitleStyle.Render("Command Breakdown")
	}
	lines = append(lines, titleLabel)
	lines = append(lines, "")

	maxTokenLen := 0
	for _, c := range result.Components {
		if len(c.Token) > maxTokenLen {
			maxTokenLen = len(c.Token)
		}
	}
	if maxTokenLen > 30 {
		maxTokenLen = 30
	}

	prevIsBinary := false
	for _, c := range result.Components {
		token := c.Token
		if len(token) > 30 {
			token = token[:27] + "..."
		}

		indent := "  "
		if c.Type == "binary" && !prevIsBinary {
			indent = ""
		}

		var descStyled string
		switch c.Danger {
		case explain.Destructive:
			marker := ErrorStyle.Render("[!] ")
			descStyled = marker + ErrorStyle.Render(c.Description)
		case explain.Caution:
			marker := WarningStyle.Render("[~] ")
			descStyled = marker + WarningStyle.Render(c.Description)
		default:
			descStyled = MetadataStyle.Render(c.Description)
		}

		padding := strings.Repeat(" ", max(1, maxTokenLen+2-len(token)))
		tokenStyled := CommandStyle.Render(indent + token)

		lines = append(lines, tokenStyled+padding+MetadataStyle.Render("<- ")+descStyled)

		prevIsBinary = c.Type == "binary"
	}

	if showWarnings && len(result.Warnings) > 0 {
		lines = append(lines, "")
		for _, w := range result.Warnings {
			switch w.Level {
			case explain.Destructive:
				lines = append(lines, ErrorStyle.Render("[!] "+w.Message))
			case explain.Caution:
				lines = append(lines, WarningStyle.Render("[~] "+w.Message))
			}
		}
	}

	if showWarnings && len(result.Suggestions) > 0 {
		lines = append(lines, "")
		for _, s := range result.Suggestions {
			lines = append(lines, HintStyle.Render("Tip: "+s))
		}
	}

	content := strings.Join(lines, "\n")
	return BorderStyle.Render(content)
}

func RenderStats(totalCommands, uniqueCommands, captureDays, period int, topCmds []StatsCommand, categories []StatsCategory, rareCmds []StatsCommand, streak *vault.StreakInfo) string {
	var lines []string

	lines = append(lines, TitleStyle.Render("Recall Stats"))
	lines = append(lines, "")

	lines = append(lines, MetadataStyle.Render(fmt.Sprintf(
		"  Vault: %s commands captured | %s unique patterns",
		CommandStyle.Render(fmt.Sprintf("%d", totalCommands)),
		CommandStyle.Render(fmt.Sprintf("%d", uniqueCommands)),
	)))
	if captureDays > 0 {
		lines = append(lines, MetadataStyle.Render(fmt.Sprintf(
			"  Capture period: %s",
			HintStyle.Render(fmt.Sprintf("%d days", captureDays)),
		)))
	}

	// Streak display
	if streak != nil && streak.CurrentStreak > 0 {
		streakLine := fmt.Sprintf("  %s %s-day streak!",
			streak.StreakEmoji,
			CommandStyle.Render(fmt.Sprintf("%d", streak.CurrentStreak)),
		)
		if streak.LongestStreak > streak.CurrentStreak {
			streakLine += MetadataStyle.Render(fmt.Sprintf("  (best: %d days)", streak.LongestStreak))
		}
		lines = append(lines, SuccessStyle.Render(streakLine))
	}
	if streak != nil && streak.TodayCount > 0 {
		lines = append(lines, MetadataStyle.Render(fmt.Sprintf(
			"  Today: %s commands captured",
			HintStyle.Render(fmt.Sprintf("%d", streak.TodayCount)),
		)))
	}

	if len(topCmds) > 0 {
		lines = append(lines, "")
		periodLabel := "this week"
		if period != 7 {
			periodLabel = fmt.Sprintf("last %d days", period)
		}
		lines = append(lines, StatsHeaderStyle.Render(fmt.Sprintf("  Top Commands (%s)", periodLabel)))
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
			bar := FrequencyBarStyle.Render(strings.Repeat("█", barLen))

			freqStr := AccentStyle.Render(fmt.Sprintf("(%d)", cmd.Frequency))
			lines = append(lines, fmt.Sprintf("  %s %s %s %s",
				DimStyle.Render(fmt.Sprintf("%2d.", i+1)),
				NormalItemStyle.Render(fmt.Sprintf("%-52s", raw)),
				freqStr,
				bar,
			))
		}
	}

	if len(categories) > 0 {
		lines = append(lines, "")
		lines = append(lines, StatsHeaderStyle.Render("  Top Categories"))
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
			catParts = append(catParts, CategoryStyle.Render(fmt.Sprintf("%s: %.0f%%", c.Category, pct)))
		}
		lines = append(lines, "  "+strings.Join(catParts, "  "))
	}

	if len(rareCmds) > 0 {
		lines = append(lines, "")
		lines = append(lines, StatsHeaderStyle.Render("  Rare but Valuable"))
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
				DimStyle.Render(fmt.Sprintf("%2d.", i+1)),
				NormalItemStyle.Render(raw),
				DimStyle.Render(fmt.Sprintf("(used %s)", freqLabel)),
			))
		}
	}

	content := strings.Join(lines, "\n")
	return BorderStyle.Render(content)
}

// StatsCommand is a simplified command struct for rendering.
type StatsCommand struct {
	Raw       string
	Frequency int
}

// StatsCategory is a simplified category struct for rendering.
type StatsCategory struct {
	Category       string
	Count          int
	TotalFrequency int
}

func RenderAliasSuggestions(suggestions []AliasSuggestion) string {
	var lines []string

	lines = append(lines, TitleStyle.Render("Suggested Aliases"))
	lines = append(lines, "")

	for _, s := range suggestions {
		raw := s.Command
		if len(raw) > 55 {
			raw = raw[:52] + "..."
		}

		lines = append(lines, MetadataStyle.Render(fmt.Sprintf("  You type this %s:",
			AccentStyle.Render(fmt.Sprintf("%dx", s.Frequency)),
		)))
		lines = append(lines, "    "+CommandStyle.Render(raw))
		lines = append(lines, HintStyle.Render(fmt.Sprintf("  -> Suggested alias: %s",
			SuccessStyle.Render(s.Alias),
		)))
		lines = append(lines, "")
	}

	lines = append(lines, DimStyle.Render("  Add to your shell config:"))
	for _, s := range suggestions {
		lines = append(lines, DimStyle.Render(fmt.Sprintf("    alias %s='%s'", s.Alias, s.Command)))
	}

	return BorderStyle.Render(strings.Join(lines, "\n"))
}

type AliasSuggestion struct {
	Command   string
	Alias     string
	Frequency int
}

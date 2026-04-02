package ui

import (
	"fmt"
	"strings"

	"github.com/Om-Rohilla/recall/internal/vault"
	"github.com/charmbracelet/lipgloss"
)

// ShowWrapped beautifully renders the user's weekly terminal stats to stdout.
func ShowWrapped(stats *vault.WrappedStats) {
	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#FF79C6")). // Pink
		MarginBottom(1).
		Render("🎉 YOUR TERMINAL WRAPPED 🎉")

	boxStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#bd93f9")). // Purple
		Padding(1, 3).
		MarginTop(1).
		MarginBottom(1)

	var sb strings.Builder
	sb.WriteString(titleStyle + "\n")

	if stats.TotalThisWeek == 0 {
		sb.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("#ffb86c")).Render("You didn't type any commands this week! Time to code?"))
		fmt.Println(boxStyle.Render(sb.String()))
		return
	}

	sb.WriteString(fmt.Sprintf("🚀 You fired off %s commands this week.\n", lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#50fa7b")).Render(fmt.Sprintf("%d", stats.TotalThisWeek))))
	
	if stats.TotalGrowth > 0 {
		sb.WriteString(fmt.Sprintf("   📈 That's a %s from last week! (Flow state activated)\n", lipgloss.NewStyle().Foreground(lipgloss.Color("#50fa7b")).Render(fmt.Sprintf("%d%% increase", stats.TotalGrowth))))
	} else if stats.TotalGrowth < 0 {
		sb.WriteString(fmt.Sprintf("   📉 That's a %s. Touching grass?\n", lipgloss.NewStyle().Foreground(lipgloss.Color("#ffb86c")).Render(fmt.Sprintf("%d%% drop", -stats.TotalGrowth))))
	}
	sb.WriteString("\n")

	sb.WriteString("🏆 Your Top 3 Most Used Commands:\n")
	for i, cmd := range stats.TopCommands {
		medal := "🥉"
		if i == 0 {
			medal = "🥇"
		} else if i == 1 {
			medal = "🥈"
		}
		sb.WriteString(fmt.Sprintf("   %s %s (%d times)\n", medal, lipgloss.NewStyle().Foreground(lipgloss.Color("#f1fa8c")).Render(cmd.Raw), cmd.Frequency))
	}
	sb.WriteString("\n")

	if stats.TopCategory != "" {
		sb.WriteString(fmt.Sprintf("💻 Top Vibe: You spent the most time grinding in [%s].\n", lipgloss.NewStyle().Foreground(lipgloss.Color("#8be9fd")).Bold(true).Render(strings.ToUpper(stats.TopCategory))))
	}

	sb.WriteString(fmt.Sprintf("🔋 Prime Time: Your busiest hour was %s UTC on %s.\n", 
		lipgloss.NewStyle().Foreground(lipgloss.Color("#ffb86c")).Render(fmt.Sprintf("%02d:00", stats.BusiestHour)),
		lipgloss.NewStyle().Foreground(lipgloss.Color("#ffb86c")).Render(stats.BusiestDay.String())))
	
	if stats.MergeConflictsFixed > 0 {
		sb.WriteString(fmt.Sprintf("🛡️  Survival: You successfully survived %d Git merge conflicts.\n", stats.MergeConflictsFixed))
	} else {
		sb.WriteString("🛡️  Survival: 0 Git merge conflicts. (You got lucky).\n")
	}

	fmt.Println(boxStyle.Render(sb.String()))
}

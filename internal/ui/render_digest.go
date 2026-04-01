package ui

import (
	"fmt"
	"strings"

	"github.com/Om-Rohilla/recall/internal/vault"
)

// DigestData holds all data needed to render a weekly digest.
type DigestData struct {
	NewCommands    []vault.Command
	TopCommands    []vault.Command
	NewCategories  []string
	Streak         *vault.StreakInfo
	Aliases        []AliasSuggestion
	TotalCommands  int
	UniqueCommands int
	PeriodDays     int
}

// RenderDigest produces a formatted weekly digest output.
func RenderDigest(d DigestData) string {
	var b strings.Builder

	// Header
	b.WriteString(TitleStyle.Render("  📊 Weekly Digest"))
	b.WriteString("\n")
	b.WriteString(MetadataStyle.Render(fmt.Sprintf("  Last %d days | %d total commands | %d unique",
		d.PeriodDays, d.TotalCommands, d.UniqueCommands)))
	b.WriteString("\n\n")

	// Streak
	if d.Streak != nil {
		b.WriteString(renderDigestSection("🔥 Streak", func() string {
			var lines []string
			lines = append(lines, fmt.Sprintf("  %s  %d-day streak (longest: %d)",
				d.Streak.StreakEmoji, d.Streak.CurrentStreak, d.Streak.LongestStreak))
			if d.Streak.IsActiveToday {
				lines = append(lines, SuccessStyle.Render(fmt.Sprintf("  ✓ Active today (%d commands)", d.Streak.TodayCount)))
			} else {
				lines = append(lines, WarningStyle.Render("  ○ Not yet active today — keep the streak alive!"))
			}
			return strings.Join(lines, "\n")
		}))
	}

	// New commands learned
	b.WriteString(renderDigestSection("📚 New Commands Learned", func() string {
		if len(d.NewCommands) == 0 {
			return DimStyle.Render("  No new commands this period.")
		}
		var lines []string
		limit := 10
		if len(d.NewCommands) < limit {
			limit = len(d.NewCommands)
		}
		for i := 0; i < limit; i++ {
			cmd := d.NewCommands[i]
			cat := ""
			if cmd.Category != "" {
				cat = DimStyle.Render(fmt.Sprintf(" [%s %s]", CategoryIcon(cmd.Category), cmd.Category))
			}
			lines = append(lines, fmt.Sprintf("  %s %s%s",
				AccentStyle.Render("•"),
				CommandStyle.Render(truncate(cmd.Raw, 60)),
				cat,
			))
		}
		if len(d.NewCommands) > limit {
			lines = append(lines, DimStyle.Render(fmt.Sprintf("  ... and %d more", len(d.NewCommands)-limit)))
		}
		return strings.Join(lines, "\n")
	}))

	// Most-used this period
	b.WriteString(renderDigestSection("⚡ Most Used", func() string {
		if len(d.TopCommands) == 0 {
			return DimStyle.Render("  No command activity this period.")
		}
		var lines []string
		for i, cmd := range d.TopCommands {
			if i >= 5 {
				break
			}
			medal := []string{"🥇", "🥈", "🥉", " 4.", " 5."}[i]
			lines = append(lines, fmt.Sprintf("  %s %s  %s",
				medal,
				CommandStyle.Render(truncate(cmd.Raw, 50)),
				DimStyle.Render(fmt.Sprintf("%dx", cmd.Frequency)),
			))
		}
		return strings.Join(lines, "\n")
	}))

	// New categories
	if len(d.NewCategories) > 0 {
		b.WriteString(renderDigestSection("🏷️  New Categories Discovered", func() string {
			var cats []string
			for _, c := range d.NewCategories {
				cats = append(cats, fmt.Sprintf("%s %s", CategoryIcon(c), c))
			}
			return "  " + HintStyle.Render(strings.Join(cats, "  •  "))
		}))
	}

	// Alias suggestions
	if len(d.Aliases) > 0 {
		b.WriteString(renderDigestSection("💡 Alias Suggestions", func() string {
			var lines []string
			limit := 5
			if len(d.Aliases) < limit {
				limit = len(d.Aliases)
			}
			for i := 0; i < limit; i++ {
				a := d.Aliases[i]
				lines = append(lines, fmt.Sprintf("  alias %s='%s'  %s",
					AccentStyle.Render(a.Alias),
					a.Command,
					DimStyle.Render(fmt.Sprintf("(%dx)", a.Frequency)),
				))
			}
			return strings.Join(lines, "\n")
		}))
	}

	return b.String()
}

func renderDigestSection(title string, content func() string) string {
	var b strings.Builder
	b.WriteString(StatsHeaderStyle.Render("  " + title))
	b.WriteString("\n")
	b.WriteString(content())
	b.WriteString("\n\n")
	return b.String()
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}

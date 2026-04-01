package ui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/Om-Rohilla/recall/internal/vault"
)

// RenderResultCard renders a single search result as a bordered card.
func RenderResultCard(r vault.SearchResult, isBest bool) string {
	label := "Match"
	if isBest {
		label = "Best Match"
	}

	confStr := fmt.Sprintf("%.0f%%", r.Confidence)
	confStyled := ConfidenceStyle(r.Confidence).Render(confStr)

	title := TitleStyle.Render(fmt.Sprintf("─ %s ", label)) + MetadataStyle.Render("(confidence: ") + confStyled + MetadataStyle.Render(")")

	command := CommandStyle.Render("  " + r.Command.Raw)

	var metaParts []string
	if r.Command.Frequency > 1 {
		metaParts = append(metaParts, fmt.Sprintf("Used %d times", r.Command.Frequency))
	}
	if !r.Command.LastSeen.IsZero() {
		metaParts = append(metaParts, "Last: "+relativeTime(r.Command.LastSeen))
	}
	if r.Command.Category != "" && r.Command.Category != "other" {
		metaParts = append(metaParts, CategoryStyle.Render(r.Command.Category))
	}

	metadata := ""
	if len(metaParts) > 0 {
		metadata = MetadataStyle.Render("  " + strings.Join(metaParts, "  |  "))
	}

	hints := ""
	if isBest {
		hints = HintStyle.Render("  💡 Use --top 3 for more results")
	}

	var lines []string
	lines = append(lines, title)
	lines = append(lines, "")
	lines = append(lines, command)
	if metadata != "" {
		lines = append(lines, "")
		lines = append(lines, metadata)
	}
	if hints != "" {
		lines = append(lines, "")
		lines = append(lines, hints)
	}

	content := strings.Join(lines, "\n")

	card := BorderStyle.Render(content)

	return card
}

// RenderResultList renders multiple search results.
func RenderResultList(results []vault.SearchResult) string {
	var cards []string
	for i, r := range results {
		cards = append(cards, RenderResultCard(r, i == 0))
	}
	return lipgloss.JoinVertical(lipgloss.Left, cards...)
}

// RenderNoResults renders the no-results message.
func RenderNoResults(query string) string {
	title := TitleStyle.Render("No results found")
	msg := MetadataStyle.Render(fmt.Sprintf("  No commands matching \"%s\"", query))

	tips := []string{
		HintStyle.Render("  Tips:"),
		MetadataStyle.Render("    - Try different keywords"),
		MetadataStyle.Render("    - Import your history: recall import-history"),
		MetadataStyle.Render("    - Use broader terms"),
	}

	lines := append([]string{title, "", msg, ""}, tips...)
	content := strings.Join(lines, "\n")

	return BorderStyle.Render(content)
}

func relativeTime(t time.Time) string {
	diff := time.Since(t)

	switch {
	case diff < time.Minute:
		return "just now"
	case diff < time.Hour:
		mins := int(diff.Minutes())
		if mins == 1 {
			return "1 minute ago"
		}
		return fmt.Sprintf("%d minutes ago", mins)
	case diff < 24*time.Hour:
		hours := int(diff.Hours())
		if hours == 1 {
			return "1 hour ago"
		}
		return fmt.Sprintf("%d hours ago", hours)
	case diff < 7*24*time.Hour:
		days := int(diff.Hours() / 24)
		if days == 1 {
			return "yesterday"
		}
		return fmt.Sprintf("%d days ago", days)
	case diff < 30*24*time.Hour:
		weeks := int(diff.Hours() / 24 / 7)
		if weeks == 1 {
			return "1 week ago"
		}
		return fmt.Sprintf("%d weeks ago", weeks)
	default:
		months := int(diff.Hours() / 24 / 30)
		if months == 1 {
			return "1 month ago"
		}
		return fmt.Sprintf("%d months ago", months)
	}
}

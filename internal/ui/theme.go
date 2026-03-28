package ui

import (
	"os"

	"github.com/charmbracelet/lipgloss"
)

var noColor = os.Getenv("NO_COLOR") != ""

var (
	ColorPrimary   = lipgloss.Color("#7C3AED") // purple
	ColorSecondary = lipgloss.Color("#06B6D4") // cyan
	ColorSuccess   = lipgloss.Color("#10B981") // green
	ColorWarning   = lipgloss.Color("#F59E0B") // amber
	ColorDanger    = lipgloss.Color("#EF4444") // red
	ColorMuted     = lipgloss.Color("#6B7280") // gray
	ColorText      = lipgloss.Color("#E5E7EB") // light gray
	ColorBright    = lipgloss.Color("#F9FAFB") // white
)

var (
	BorderStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(ColorPrimary).
			Padding(1, 2)

	TitleStyle = lipgloss.NewStyle().
			Foreground(ColorPrimary).
			Bold(true)

	CommandStyle = lipgloss.NewStyle().
			Foreground(ColorBright).
			Bold(true)

	MetadataStyle = lipgloss.NewStyle().
			Foreground(ColorMuted)

	ConfidenceHighStyle = lipgloss.NewStyle().
				Foreground(ColorSuccess).
				Bold(true)

	ConfidenceMedStyle = lipgloss.NewStyle().
				Foreground(ColorWarning).
				Bold(true)

	ConfidenceLowStyle = lipgloss.NewStyle().
				Foreground(ColorDanger).
				Bold(true)

	HintStyle = lipgloss.NewStyle().
			Foreground(ColorSecondary)

	SuccessStyle = lipgloss.NewStyle().
			Foreground(ColorSuccess)

	WarningStyle = lipgloss.NewStyle().
			Foreground(ColorWarning)

	ErrorStyle = lipgloss.NewStyle().
			Foreground(ColorDanger)

	CategoryStyle = lipgloss.NewStyle().
			Foreground(ColorSecondary).
			Italic(true)
)

func init() {
	if noColor {
		// Reset all styles to plain text
		BorderStyle = lipgloss.NewStyle().Padding(1, 2)
		TitleStyle = lipgloss.NewStyle()
		CommandStyle = lipgloss.NewStyle()
		MetadataStyle = lipgloss.NewStyle()
		ConfidenceHighStyle = lipgloss.NewStyle()
		ConfidenceMedStyle = lipgloss.NewStyle()
		ConfidenceLowStyle = lipgloss.NewStyle()
		HintStyle = lipgloss.NewStyle()
		SuccessStyle = lipgloss.NewStyle()
		WarningStyle = lipgloss.NewStyle()
		ErrorStyle = lipgloss.NewStyle()
		CategoryStyle = lipgloss.NewStyle()
	}
}

func ConfidenceStyle(confidence float64) lipgloss.Style {
	switch {
	case confidence >= 80:
		return ConfidenceHighStyle
	case confidence >= 50:
		return ConfidenceMedStyle
	default:
		return ConfidenceLowStyle
	}
}

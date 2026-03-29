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
	ColorDim       = lipgloss.Color("#4B5563") // dark gray
	ColorAccent    = lipgloss.Color("#8B5CF6") // light purple
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

	// TUI-specific styles
	ActiveTabStyle = lipgloss.NewStyle().
			Foreground(ColorBright).
			Background(ColorPrimary).
			Bold(true).
			Padding(0, 2)

	InactiveTabStyle = lipgloss.NewStyle().
				Foreground(ColorMuted).
				Padding(0, 2)

	SelectedItemStyle = lipgloss.NewStyle().
				Foreground(ColorBright).
				Background(ColorPrimary).
				Bold(true)

	NormalItemStyle = lipgloss.NewStyle().
			Foreground(ColorText)

	StatusBarStyle = lipgloss.NewStyle().
			Foreground(ColorMuted).
			Background(lipgloss.Color("#1F2937"))

	SearchInputStyle = lipgloss.NewStyle().
				Foreground(ColorBright).
				Border(lipgloss.RoundedBorder()).
				BorderForeground(ColorSecondary).
				Padding(0, 1)

	BadgeStyle = lipgloss.NewStyle().
			Foreground(ColorBright).
			Background(ColorPrimary).
			Bold(true).
			Padding(0, 1)

	DimStyle = lipgloss.NewStyle().
			Foreground(ColorDim)

	AccentStyle = lipgloss.NewStyle().
			Foreground(ColorAccent)

	FrequencyBarStyle = lipgloss.NewStyle().
				Foreground(ColorSuccess)

	StatsHeaderStyle = lipgloss.NewStyle().
				Foreground(ColorPrimary).
				Bold(true).
				Border(lipgloss.NormalBorder(), false, false, true, false).
				BorderForeground(ColorDim)
)

func init() {
	if noColor {
		plain := lipgloss.NewStyle()
		BorderStyle = lipgloss.NewStyle().Padding(1, 2)
		TitleStyle = plain
		CommandStyle = plain
		MetadataStyle = plain
		ConfidenceHighStyle = plain
		ConfidenceMedStyle = plain
		ConfidenceLowStyle = plain
		HintStyle = plain
		SuccessStyle = plain
		WarningStyle = plain
		ErrorStyle = plain
		CategoryStyle = plain
		ActiveTabStyle = plain
		InactiveTabStyle = plain
		SelectedItemStyle = plain
		NormalItemStyle = plain
		StatusBarStyle = plain
		SearchInputStyle = plain
		BadgeStyle = plain
		DimStyle = plain
		AccentStyle = plain
		FrequencyBarStyle = plain
		StatsHeaderStyle = plain
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

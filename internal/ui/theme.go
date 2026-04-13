package ui

import (
	"os"

	"github.com/charmbracelet/lipgloss"
)

var noColor = os.Getenv("NO_COLOR") != ""

var (
	ColorPrimary   = lipgloss.Color("#BD93F9") // Premium Purple
	ColorSecondary = lipgloss.Color("#8BE9FD") // Cyan Accent
	ColorSuccess   = lipgloss.Color("#50FA7B") // Neon Green
	ColorWarning   = lipgloss.Color("#FFB86C") // Amber/Orange
	ColorDanger    = lipgloss.Color("#FF5555") // Red
	ColorMuted     = lipgloss.Color("#6272A4") // Muted Blue-Gray
	ColorText      = lipgloss.Color("#F8F8F2") // Off-White
	ColorBright    = lipgloss.Color("#FFFFFF") // Pure White
	ColorDim       = lipgloss.Color("#44475A") // Dark Gray/Selection
	ColorAccent    = lipgloss.Color("#FF79C6") // Pink Accent
)

var (
	PanelStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(ColorDim).
			Padding(1, 4).
			Margin(1, 0)

	BorderStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(ColorPrimary).
			Padding(1, 3)

	TitleStyle = lipgloss.NewStyle().
			Foreground(ColorAccent).
			Bold(true).
			MarginBottom(1)

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
				Foreground(ColorText).
				Background(ColorDim).
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
		PanelStyle = lipgloss.NewStyle().Padding(1, 2).Margin(1, 0)
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

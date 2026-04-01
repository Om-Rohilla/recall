package ui

import (
	"fmt"
	"strings"

	"github.com/atotto/clipboard"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/Om-Rohilla/recall/internal/vault"
)

// ResultBrowserModel is an interactive Bubbletea model for browsing search results.
// Supports [c] Copy, [Tab] next result, [Enter] select, [Esc/q] quit.
type ResultBrowserModel struct {
	results  []vault.SearchResult
	cursor   int
	copied   string
	quitting bool
	width    int
}

// NewResultBrowser creates a new interactive result browser.
func NewResultBrowser(results []vault.SearchResult) ResultBrowserModel {
	return ResultBrowserModel{
		results: results,
		cursor:  0,
		width:   80,
	}
}

func (m ResultBrowserModel) Init() tea.Cmd {
	return nil
}

func (m ResultBrowserModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "q", "esc", "ctrl+c":
			m.quitting = true
			return m, tea.Quit

		case "enter":
			if len(m.results) > 0 && m.cursor < len(m.results) {
				raw := m.results[m.cursor].Command.Raw
				_ = clipboard.WriteAll(raw)
				m.copied = raw
				m.quitting = true
				return m, tea.Quit
			}

		case "c":
			if len(m.results) > 0 && m.cursor < len(m.results) {
				raw := m.results[m.cursor].Command.Raw
				_ = clipboard.WriteAll(raw)
				m.copied = raw
				m.quitting = true
				return m, tea.Quit
			}

		case "tab", "down", "j":
			if len(m.results) > 0 {
				m.cursor = (m.cursor + 1) % len(m.results)
			}
			return m, nil

		case "shift+tab", "up", "k":
			if len(m.results) > 0 {
				m.cursor--
				if m.cursor < 0 {
					m.cursor = len(m.results) - 1
				}
			}
			return m, nil
		}
	}

	return m, nil
}

func (m ResultBrowserModel) View() string {
	if m.quitting {
		return ""
	}

	var b strings.Builder
	b.WriteString("\n")

	for i, r := range m.results {
		isSelected := i == m.cursor
		isBest := i == 0

		label := "Match"
		if isBest {
			label = "Best Match"
		}

		confStr := fmt.Sprintf("%.0f%%", r.Confidence)
		confStyled := ConfidenceStyle(r.Confidence).Render(confStr)

		title := TitleStyle.Render(fmt.Sprintf("─ %s ", label)) +
			MetadataStyle.Render("(confidence: ") + confStyled + MetadataStyle.Render(")")

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

		var lines []string
		lines = append(lines, title)
		lines = append(lines, "")
		lines = append(lines, command)
		if metadata != "" {
			lines = append(lines, "")
			lines = append(lines, metadata)
		}
		lines = append(lines, "")

		content := strings.Join(lines, "\n")

		if isSelected {
			card := SelectedBorderStyle.Render(content)
			b.WriteString(card + "\n")
		} else {
			card := BorderStyle.Render(content)
			b.WriteString(card + "\n")
		}
	}

	b.WriteString("\n")
	b.WriteString("  " + HintStyle.Render("[c/Enter] Copy  [Tab/↓] Next  [Shift+Tab/↑] Prev  [q/Esc] Quit") + "\n")
	b.WriteString("\n")

	return b.String()
}

// CopiedCommand returns the command that was copied, or "" if none.
func (m ResultBrowserModel) CopiedCommand() string {
	return m.copied
}

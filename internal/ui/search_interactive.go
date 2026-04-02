package ui

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/atotto/clipboard"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/Om-Rohilla/recall/internal/vault"
)

// SearchAction represents the action the user chose on a search result.
type SearchAction int

const (
	ActionNone    SearchAction = iota
	ActionExecute              // Enter — run the command
	ActionCopy                 // c — copy to clipboard
	ActionEdit                 // e — open in $EDITOR
	ActionExplain              // x — explain the command
)

// SearchInteractiveResult holds the outcome of the interactive search.
type SearchInteractiveResult struct {
	Command vault.SearchResult
	Action  SearchAction
}

// SearchInteractiveModel is a Bubbletea model for interactive search result selection.
type SearchInteractiveModel struct {
	results  []vault.SearchResult
	cursor   int
	width    int
	height   int
	quitting bool
	outcome  *SearchInteractiveResult
	message  string // transient status message (e.g., "Copied!")
}

// NewSearchInteractive creates a new interactive result picker.
func NewSearchInteractive(results []vault.SearchResult) SearchInteractiveModel {
	return SearchInteractiveModel{
		results: results,
		cursor:  0,
		width:   80,
		height:  24,
	}
}

func (m SearchInteractiveModel) Init() tea.Cmd {
	return nil
}

func (m SearchInteractiveModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "q", "esc", "ctrl+c":
			m.quitting = true
			return m, tea.Quit

		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
				m.message = ""
			}
			return m, nil

		case "down", "j":
			if m.cursor < len(m.results)-1 {
				m.cursor++
				m.message = ""
			}
			return m, nil

		case "tab":
			if len(m.results) > 0 {
				m.cursor = (m.cursor + 1) % len(m.results)
				m.message = ""
			}
			return m, nil

		case "shift+tab":
			if len(m.results) > 0 {
				m.cursor--
				if m.cursor < 0 {
					m.cursor = len(m.results) - 1
				}
				m.message = ""
			}
			return m, nil

		case "enter":
			if len(m.results) > 0 && m.cursor < len(m.results) {
				m.outcome = &SearchInteractiveResult{
					Command: m.results[m.cursor],
					Action:  ActionExecute,
				}
				m.quitting = true
				return m, tea.Quit
			}
			return m, nil

		case "c":
			if len(m.results) > 0 && m.cursor < len(m.results) {
				cmd := m.results[m.cursor].Command.Raw
				if err := clipboard.WriteAll(cmd); err != nil {
					m.message = ErrorStyle.Render("  ✗ Could not copy to clipboard")
				} else {
					m.message = SuccessStyle.Render("  ✓ Copied to clipboard!")
				}
			}
			return m, nil

		case "e":
			if len(m.results) > 0 && m.cursor < len(m.results) {
				m.outcome = &SearchInteractiveResult{
					Command: m.results[m.cursor],
					Action:  ActionEdit,
				}
				m.quitting = true
				return m, tea.Quit
			}
			return m, nil

		case "x":
			if len(m.results) > 0 && m.cursor < len(m.results) {
				m.outcome = &SearchInteractiveResult{
					Command: m.results[m.cursor],
					Action:  ActionExplain,
				}
				m.quitting = true
				return m, tea.Quit
			}
			return m, nil
		}
	}

	return m, nil
}

func (m SearchInteractiveModel) View() string {
	if m.quitting {
		return ""
	}

	var b strings.Builder

	b.WriteString("\n")
	b.WriteString("  " + TitleStyle.Render("Search Results") + "\n\n")

	for i, r := range m.results {
		isSelected := i == m.cursor

		confStr := fmt.Sprintf("%.0f%%", r.Confidence)
		confStyled := ConfidenceStyle(r.Confidence).Render(confStr)

		// Micro-Interaction: ✨ Sparkle heavily on near-perfect matches (Cuteness Engineering)
		if r.Confidence > 94 {
			confStyled = lipgloss.NewStyle().Foreground(lipgloss.Color("#FFB86C")).Bold(true).Render("✨ " + confStr)
		}

		raw := r.Command.Raw
		maxLen := m.width - 30
		if maxLen < 30 {
			maxLen = 30
		}
		if len(raw) > maxLen {
			raw = raw[:maxLen-3] + "..."
		}

		icon := CategoryIcon(r.Command.Category)

		if isSelected {
			prefix := SelectedItemStyle.Render(" ▸ " + icon + " ")
			
			// Neon Border Effect around selected text if perfectly matched
			cmdStyle := CommandStyle
			if r.Confidence > 94 {
				cmdStyle = cmdStyle.Copy().Foreground(lipgloss.Color("#FF79C6")).Bold(true).Underline(true)
			}
			cmdText := cmdStyle.Render(raw)

			b.WriteString(prefix + cmdText + "  " + confStyled + "\n")

			// Show metadata for selected item
			var meta []string
			if r.Command.Frequency > 1 {
				meta = append(meta, fmt.Sprintf("⌨️  Used %dx", r.Command.Frequency))
			}
			if r.Command.Category != "" && r.Command.Category != "other" {
				meta = append(meta, CategoryStyle.Render(strings.ToUpper(r.Command.Category)))
			}
			if !r.Command.LastSeen.IsZero() {
				meta = append(meta, relativeTime(r.Command.LastSeen))
			}
			if r.MatchType != "" {
				meta = append(meta, DimStyle.Render("via "+r.MatchType))
			}
			if len(meta) > 0 {
				// Indented metadata under the command
				b.WriteString("       " + MetadataStyle.Render(strings.Join(meta, "  │  ")) + "\n")
			}
		} else {
			b.WriteString("   " + icon + " " + NormalItemStyle.Render(raw) + "  " + confStyled + "\n")
		}
	}

	// Status message (e.g., "Copied!")
	if m.message != "" {
		b.WriteString("\n" + m.message + "\n")
	}

	// Action bar
	b.WriteString("\n")
	b.WriteString("  " + HintStyle.Render("[Enter]") + " Execute  ")
	b.WriteString(HintStyle.Render("[c]") + " Copy  ")
	b.WriteString(HintStyle.Render("[e]") + " Edit  ")
	b.WriteString(HintStyle.Render("[x]") + " Explain  ")
	b.WriteString(HintStyle.Render("[↑/↓]") + " Navigate  ")
	b.WriteString(HintStyle.Render("[Esc]") + " Quit\n")

	return b.String()
}

// Outcome returns the user's chosen action, or nil if they quit.
func (m SearchInteractiveModel) Outcome() *SearchInteractiveResult {
	return m.outcome
}

// ExecuteCommand runs a command in the user's shell.
func ExecuteCommand(command string) error {
	shell := os.Getenv("SHELL")
	if shell == "" {
		shell = "/bin/sh"
	}

	cmd := exec.Command(shell, "-c", command)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}

// EditCommand opens the command in $EDITOR and returns the edited result.
func EditCommand(command string) (string, error) {
	editor := os.Getenv("VISUAL")
	if editor == "" {
		editor = os.Getenv("EDITOR")
	}
	if editor == "" {
		editor = "vi"
	}

	// Write command to a temp file
	tmpFile, err := os.CreateTemp("", "recall-edit-*.sh")
	if err != nil {
		return "", fmt.Errorf("creating temp file: %w", err)
	}
	defer os.Remove(tmpFile.Name())

	if _, err := tmpFile.WriteString(command); err != nil {
		tmpFile.Close()
		return "", fmt.Errorf("writing to temp file: %w", err)
	}
	tmpFile.Close()

	// Open editor
	editorCmd := exec.Command(editor, tmpFile.Name())
	editorCmd.Stdin = os.Stdin
	editorCmd.Stdout = os.Stdout
	editorCmd.Stderr = os.Stderr

	if err := editorCmd.Run(); err != nil {
		return "", fmt.Errorf("running editor: %w", err)
	}

	// Read back
	edited, err := os.ReadFile(tmpFile.Name())
	if err != nil {
		return "", fmt.Errorf("reading edited file: %w", err)
	}

	return strings.TrimSpace(string(edited)), nil
}

// CategoryIcon returns an emoji icon for a command category.
func CategoryIcon(category string) string {
	icons := map[string]string{
		"git":            "🐙",
		"docker":         "🐳",
		"kubernetes":     "☸",
		"filesystem":     "📁",
		"network":        "🌐",
		"process":        "⚙️",
		"archive":        "📦",
		"text":           "📝",
		"package":        "📚",
		"system":         "🖥️",
		"cloud":          "☁️",
		"build":          "🔨",
		"language":       "💻",
		"infrastructure": "🏗️",
		"other":          "📌",
	}
	if icon, ok := icons[category]; ok {
		return icon
	}
	return "📌"
}

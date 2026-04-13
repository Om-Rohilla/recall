package ui

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"

	appctx "github.com/Om-Rohilla/recall/internal/context"
	"github.com/Om-Rohilla/recall/internal/intelligence"
	"github.com/Om-Rohilla/recall/internal/vault"
)

const searchDebounceMs = 150

type SearchModel struct {
	store   *vault.Store
	input   textinput.Model
	results []vault.SearchResult
	cursor  int
	width   int
	height  int

	selected     *vault.SearchResult
	quitting     bool
	ready        bool
	kbPath       string
	pendingQuery string
	debounceID   int
}

type debounceTickMsg struct {
	id    int
	query string
}

func NewSearchModel(store *vault.Store, kbPath string) SearchModel {
	ti := textinput.New()
	ti.Placeholder = "Describe what you need..."
	ti.Focus()
	ti.CharLimit = 256
	ti.Width = 60

	return SearchModel{
		store:  store,
		input:  ti,
		kbPath: kbPath,
		ready:  true,
		width:  80,
		height: 24,
	}
}

type searchResultsMsg struct {
	results []vault.SearchResult
}

func doSearch(store *vault.Store, query string, kbPath string) tea.Cmd {
	return func() tea.Msg {
		if strings.TrimSpace(query) == "" {
			return searchResultsMsg{}
		}

		cwd, _ := os.Getwd()
		currentCtx := appctx.Detect(cwd)

		opts := intelligence.SearchOptions{
			Limit:  10,
			KBPath: kbPath,
		}

		engine := intelligence.NewEngine(store)
		results, err := engine.Search(query, currentCtx, opts)
		if err != nil {
			return searchResultsMsg{}
		}
		return searchResultsMsg{results: results}
	}
}

func (m SearchModel) Init() tea.Cmd {
	return textinput.Blink
}

func (m SearchModel) scheduleDebounce(query string) tea.Cmd {
	id := m.debounceID
	return tea.Tick(time.Duration(searchDebounceMs)*time.Millisecond, func(t time.Time) tea.Msg {
		return debounceTickMsg{id: id, query: query}
	})
}

func (m SearchModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case debounceTickMsg:
		if msg.id == m.debounceID && msg.query == m.input.Value() {
			return m, doSearch(m.store, msg.query, m.kbPath)
		}
		return m, nil

	case searchResultsMsg:
		m.results = msg.results
		m.cursor = 0
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "esc":
			m.quitting = true
			return m, tea.Quit

		case "enter":
			if len(m.results) > 0 && m.cursor < len(m.results) {
				r := m.results[m.cursor]
				m.selected = &r
				m.quitting = true
				return m, tea.Quit
			}
			return m, doSearch(m.store, m.input.Value(), m.kbPath)

		case "tab":
			if len(m.results) > 0 {
				m.cursor = (m.cursor + 1) % len(m.results)
			}
			return m, nil

		case "shift+tab":
			if len(m.results) > 0 {
				m.cursor--
				if m.cursor < 0 {
					m.cursor = len(m.results) - 1
				}
			}
			return m, nil

		case "up":
			if m.cursor > 0 {
				m.cursor--
			}
			return m, nil

		case "down":
			if m.cursor < len(m.results)-1 {
				m.cursor++
			}
			return m, nil

		default:
			var cmd tea.Cmd
			m.input, cmd = m.input.Update(msg)
			m.debounceID++
			m.pendingQuery = m.input.Value()
			return m, tea.Batch(cmd, m.scheduleDebounce(m.pendingQuery))
		}
	}

	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	return m, cmd
}

func (m SearchModel) View() string {
	if m.quitting {
		return ""
	}

	var b strings.Builder

	b.WriteString(TitleStyle.Render("Recall Search") + "\n")
	b.WriteString("\n")
	b.WriteString(SearchInputStyle.Render(m.input.View()) + "\n")
	b.WriteString("\n")

	if len(m.results) == 0 {
		if m.input.Value() != "" {
			b.WriteString("  " + DimStyle.Render("No results found. Try different keywords.") + "\n")
		} else {
			b.WriteString("  " + DimStyle.Render("Type to search your command vault...") + "\n")
		}
	} else {
		maxResults := m.height - 10
		if maxResults < 3 {
			maxResults = 3
		}
		if maxResults > len(m.results) {
			maxResults = len(m.results)
		}

		for i := 0; i < maxResults; i++ {
			r := m.results[i]
			isSelected := i == m.cursor

			confStr := fmt.Sprintf("%.0f%%", r.Confidence)
			confStyled := ConfidenceStyle(r.Confidence).Render(confStr)

			raw := r.Command.Raw
			maxLen := m.width - 25
			if maxLen < 30 {
				maxLen = 30
			}
			if len(raw) > maxLen {
				raw = raw[:maxLen-3] + "..."
			}

			if isSelected {
				prefix := SelectedItemStyle.Render(" > ")
				cmdText := CommandStyle.Render(raw)
				b.WriteString(prefix + cmdText + "  " + confStyled + "\n")

				var meta []string
				if r.Command.Frequency > 1 {
					meta = append(meta, fmt.Sprintf("Used %dx", r.Command.Frequency))
				}
				if r.Command.Category != "" {
					meta = append(meta, CategoryStyle.Render(r.Command.Category))
				}
				if !r.Command.LastSeen.IsZero() {
					meta = append(meta, relativeTime(r.Command.LastSeen))
				}
				if len(meta) > 0 {
					b.WriteString("    " + MetadataStyle.Render(strings.Join(meta, "  |  ")) + "\n")
				}
			} else {
				b.WriteString("   " + NormalItemStyle.Render(raw) + "  " + confStyled + "\n")
			}
		}
	}

	// Action bar
	b.WriteString("\n")
	actionBar := HintStyle.Render("[Enter]") + " Select  " +
		HintStyle.Render("[Tab]") + " Next  " +
		HintStyle.Render("[↑/↓]") + " Navigate  " +
		HintStyle.Render("[Esc]") + " Quit"

	b.WriteString(DimStyle.Render(strings.Repeat("─", m.width-8)) + "\n")
	b.WriteString(actionBar + "\n")

	return PanelStyle.Render(b.String())
}

func (m SearchModel) Selected() *vault.SearchResult {
	return m.selected
}

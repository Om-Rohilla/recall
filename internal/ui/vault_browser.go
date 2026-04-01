package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"

	"github.com/Om-Rohilla/recall/internal/vault"
	"github.com/Om-Rohilla/recall/pkg/logging"
)

type viewMode int

const (
	viewList viewMode = iota
	viewCategories
	viewDetails
	viewHelp
)

type sortMode int

const (
	sortRecency sortMode = iota
	sortFrequency
	sortAlpha
)

func (s sortMode) String() string {
	switch s {
	case sortRecency:
		return "recency"
	case sortFrequency:
		return "frequency"
	case sortAlpha:
		return "alpha"
	default:
		return "recency"
	}
}

type VaultBrowserModel struct {
	store      *vault.Store
	commands   []vault.Command
	filtered   []vault.Command
	categories []vault.CategoryCount

	cursor    int
	offset    int
	height    int
	width     int
	view      viewMode
	sort      sortMode
	searching bool
	searchTI  textinput.Model

	filterCategory string
	selectedCmd    *vault.Command
	selectedCtxs   []vault.Context

	confirmDelete  bool
	deleteTargetID int64

	cachedStats *vault.VaultStats

	ready   bool
	quitting bool
}

func NewVaultBrowser(store *vault.Store, category string, sortBy string) VaultBrowserModel {
	ti := textinput.New()
	ti.Placeholder = "Type to filter..."
	ti.CharLimit = 128
	ti.Width = 50

	s := sortRecency
	switch sortBy {
	case "frequency":
		s = sortFrequency
	case "alpha":
		s = sortAlpha
	}

	m := VaultBrowserModel{
		store:          store,
		searchTI:       ti,
		sort:           s,
		filterCategory: category,
		height:         24,
		width:          80,
	}
	return m
}

type commandsLoadedMsg struct {
	commands   []vault.Command
	categories []vault.CategoryCount
	err        error
}

func loadCommands(store *vault.Store, sortBy sortMode, category string) tea.Cmd {
	return func() tea.Msg {
		var cmds []vault.Command
		var loadErr error

		if category != "" {
			cmds, loadErr = store.GetCommandsByCategory(category, 500)
		} else {
			cmds, loadErr = store.GetAllCommands(sortBy.String(), 500)
		}

		cats, _ := store.GetCategories()

		return commandsLoadedMsg{commands: cmds, categories: cats, err: loadErr}
	}
}

func (m VaultBrowserModel) Init() tea.Cmd {
	return loadCommands(m.store, m.sort, m.filterCategory)
}

func (m VaultBrowserModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case commandsLoadedMsg:
		m.commands = msg.commands
		m.categories = msg.categories
		m.filtered = m.applyFilter()
		m.cachedStats, _ = m.store.GetStats()
		m.ready = true
		return m, nil

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case tea.KeyMsg:
		if m.searching {
			return m.handleSearchInput(msg)
		}
		return m.handleNormalInput(msg)
	}

	return m, nil
}

func (m VaultBrowserModel) handleSearchInput(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.searching = false
		m.searchTI.Reset()
		m.filtered = m.applyFilter()
		m.cursor = 0
		m.offset = 0
		return m, nil
	case "enter":
		m.searching = false
		return m, nil
	default:
		var cmd tea.Cmd
		m.searchTI, cmd = m.searchTI.Update(msg)
		m.filtered = m.applyFilter()
		m.cursor = 0
		m.offset = 0
		return m, cmd
	}
}

func (m VaultBrowserModel) handleNormalInput(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.confirmDelete {
		switch msg.String() {
		case "y", "Y":
			if err := m.store.DeleteCommand(m.deleteTargetID); err != nil {
				logging.Get().Warn("failed to delete command", "id", m.deleteTargetID, "error", err)
			}
			m.confirmDelete = false
			m.deleteTargetID = 0
			return m, loadCommands(m.store, m.sort, m.filterCategory)
		default:
			m.confirmDelete = false
			m.deleteTargetID = 0
			return m, nil
		}
	}

	switch msg.String() {
	case "q", "ctrl+c":
		m.quitting = true
		return m, tea.Quit

	case "?":
		if m.view == viewHelp {
			m.view = viewList
		} else {
			m.view = viewHelp
		}
		return m, nil

	case "esc":
		switch m.view {
		case viewDetails:
			m.view = viewList
			m.selectedCmd = nil
		case viewCategories:
			m.view = viewList
		case viewHelp:
			m.view = viewList
		default:
			m.quitting = true
			return m, tea.Quit
		}
		return m, nil

	case "/":
		m.searching = true
		m.searchTI.Focus()
		return m, textinput.Blink

	case "tab":
		switch m.view {
		case viewList:
			m.view = viewCategories
			m.cursor = 0
			m.offset = 0
		case viewCategories:
			m.view = viewList
			m.cursor = 0
			m.offset = 0
		}
		return m, nil

	case "up", "k":
		if m.cursor > 0 {
			m.cursor--
			if m.cursor < m.offset {
				m.offset = m.cursor
			}
		}
		return m, nil

	case "down", "j":
		maxIdx := m.maxIndex()
		if m.cursor < maxIdx {
			m.cursor++
			viewable := m.viewableRows()
			if m.cursor >= m.offset+viewable {
				m.offset = m.cursor - viewable + 1
			}
		}
		return m, nil

	case "pgup":
		viewable := m.viewableRows()
		m.cursor -= viewable
		if m.cursor < 0 {
			m.cursor = 0
		}
		m.offset = m.cursor
		return m, nil

	case "pgdown":
		viewable := m.viewableRows()
		maxIdx := m.maxIndex()
		m.cursor += viewable
		if m.cursor > maxIdx {
			m.cursor = maxIdx
		}
		if m.cursor >= m.offset+viewable {
			m.offset = m.cursor - viewable + 1
		}
		return m, nil

	case "home", "g":
		m.cursor = 0
		m.offset = 0
		return m, nil

	case "end", "G":
		maxIdx := m.maxIndex()
		m.cursor = maxIdx
		viewable := m.viewableRows()
		if m.cursor >= viewable {
			m.offset = m.cursor - viewable + 1
		}
		return m, nil

	case "enter":
		if m.view == viewCategories && len(m.categories) > 0 {
			cat := m.categories[m.cursor]
			m.filterCategory = cat.Category
			m.view = viewList
			m.cursor = 0
			m.offset = 0
			return m, loadCommands(m.store, m.sort, m.filterCategory)
		}
		return m, nil

	case "i":
		if m.view == viewList && len(m.filtered) > 0 && m.cursor < len(m.filtered) {
			cmd := m.filtered[m.cursor]
			m.selectedCmd = &cmd
			ctxs, _ := m.store.GetContextsForCommand(cmd.ID)
			m.selectedCtxs = ctxs
			m.view = viewDetails
		}
		return m, nil

	case "d":
		if m.view == viewList && len(m.filtered) > 0 && m.cursor < len(m.filtered) {
			cmd := m.filtered[m.cursor]
			m.confirmDelete = true
			m.deleteTargetID = cmd.ID
		}
		return m, nil


	case "s":
		switch m.sort {
		case sortRecency:
			m.sort = sortFrequency
		case sortFrequency:
			m.sort = sortAlpha
		case sortAlpha:
			m.sort = sortRecency
		}
		m.cursor = 0
		m.offset = 0
		return m, loadCommands(m.store, m.sort, m.filterCategory)

	case "backspace":
		if m.filterCategory != "" {
			m.filterCategory = ""
			m.cursor = 0
			m.offset = 0
			return m, loadCommands(m.store, m.sort, "")
		}
		return m, nil
	}

	return m, nil
}

func (m VaultBrowserModel) maxIndex() int {
	switch m.view {
	case viewCategories:
		if len(m.categories) == 0 {
			return 0
		}
		return len(m.categories) - 1
	default:
		if len(m.filtered) == 0 {
			return 0
		}
		return len(m.filtered) - 1
	}
}

func (m VaultBrowserModel) viewableRows() int {
	rows := m.height - 8 // header + status bar + padding
	if rows < 3 {
		rows = 3
	}
	return rows
}

func (m VaultBrowserModel) applyFilter() []vault.Command {
	query := strings.ToLower(strings.TrimSpace(m.searchTI.Value()))
	if query == "" {
		return m.commands
	}
	terms := strings.Fields(query)
	var filtered []vault.Command
	for _, cmd := range m.commands {
		cmdLower := strings.ToLower(cmd.Raw + " " + cmd.Category + " " + cmd.Binary)
		match := true
		for _, t := range terms {
			if !strings.Contains(cmdLower, t) {
				match = false
				break
			}
		}
		if match {
			filtered = append(filtered, cmd)
		}
	}
	return filtered
}

func (m VaultBrowserModel) View() string {
	if m.quitting {
		return ""
	}
	if !m.ready {
		return TitleStyle.Render("  Loading vault...")
	}

	var b strings.Builder

	// Header
	b.WriteString(m.renderHeader())
	b.WriteString("\n")

	// Search bar (if active)
	if m.searching {
		b.WriteString("  " + SearchInputStyle.Render(m.searchTI.View()))
		b.WriteString("\n")
	}

	// Main content
	switch m.view {
	case viewList:
		b.WriteString(m.renderList())
	case viewCategories:
		b.WriteString(m.renderCategories())
	case viewDetails:
		b.WriteString(m.renderDetails())
	case viewHelp:
		b.WriteString(m.renderHelp())
	}

	// Status bar
	b.WriteString("\n")
	b.WriteString(m.renderStatusBar())

	return b.String()
}

func (m VaultBrowserModel) renderHeader() string {
	total := 0
	unique := 0
	if m.cachedStats != nil {
		total = m.cachedStats.TotalCommands
		unique = m.cachedStats.UniqueCommands
	}

	title := TitleStyle.Render("  Recall Vault")
	info := MetadataStyle.Render(fmt.Sprintf("  %d commands | %d unique", total, unique))

	if m.filterCategory != "" {
		info += "  " + BadgeStyle.Render(m.filterCategory)
	}

	// Tabs
	var tabs []string
	if m.view == viewList {
		tabs = append(tabs, ActiveTabStyle.Render(" Commands "))
	} else {
		tabs = append(tabs, InactiveTabStyle.Render(" Commands "))
	}
	if m.view == viewCategories {
		tabs = append(tabs, ActiveTabStyle.Render(" Categories "))
	} else {
		tabs = append(tabs, InactiveTabStyle.Render(" Categories "))
	}

	tabRow := strings.Join(tabs, " ")
	sortLabel := DimStyle.Render(fmt.Sprintf("  sort: %s", m.sort.String()))

	return title + info + "\n" + tabRow + sortLabel
}

func (m VaultBrowserModel) renderList() string {
	if len(m.filtered) == 0 {
		return "\n" + MetadataStyle.Render("  No commands found. Import history with: recall import-history")
	}

	viewable := m.viewableRows()
	end := m.offset + viewable
	if end > len(m.filtered) {
		end = len(m.filtered)
	}

	var lines []string
	for i := m.offset; i < end; i++ {
		cmd := m.filtered[i]
		isSelected := i == m.cursor

		raw := cmd.Raw
		maxCmdLen := m.width - 30
		if maxCmdLen < 30 {
			maxCmdLen = 30
		}
		if len(raw) > maxCmdLen {
			raw = raw[:maxCmdLen-3] + "..."
		}

		freqStr := fmt.Sprintf("%dx", cmd.Frequency)
		catStr := CategoryIcon(cmd.Category) + " " + cmd.Category

		if isSelected {
			prefix := SelectedItemStyle.Render(" > ")
			cmdText := SelectedItemStyle.Render(raw)
			meta := fmt.Sprintf(" %s  %s", freqStr, catStr)
			lines = append(lines, prefix+cmdText+DimStyle.Render(meta))
		} else {
			prefix := "   "
			cmdText := NormalItemStyle.Render(raw)
			meta := DimStyle.Render(fmt.Sprintf(" %s  %s", freqStr, catStr))
			lines = append(lines, prefix+cmdText+meta)
		}
	}

	return "\n" + strings.Join(lines, "\n")
}

func (m VaultBrowserModel) renderCategories() string {
	if len(m.categories) == 0 {
		return "\n" + MetadataStyle.Render("  No categories found.")
	}

	viewable := m.viewableRows()
	end := m.offset + viewable
	if end > len(m.categories) {
		end = len(m.categories)
	}

	totalFreq := 0
	for _, c := range m.categories {
		totalFreq += c.TotalFrequency
	}
	if totalFreq == 0 {
		totalFreq = 1
	}

	var lines []string
	for i := m.offset; i < end; i++ {
		cat := m.categories[i]
		isSelected := i == m.cursor

		pct := float64(cat.TotalFrequency) / float64(totalFreq) * 100
		barLen := int(pct / 3)
		if barLen < 1 {
			barLen = 1
		}
		if barLen > 25 {
			barLen = 25
		}
		bar := FrequencyBarStyle.Render(strings.Repeat("█", barLen))

		label := fmt.Sprintf("%-2s %-13s %3d cmds  %5.1f%%  ", CategoryIcon(cat.Category), cat.Category, cat.Count, pct)

		if isSelected {
			prefix := SelectedItemStyle.Render(" > ")
			lines = append(lines, prefix+SelectedItemStyle.Render(label)+bar)
		} else {
			lines = append(lines, "   "+NormalItemStyle.Render(label)+bar)
		}
	}

	return "\n" + strings.Join(lines, "\n")
}

func (m VaultBrowserModel) renderDetails() string {
	if m.selectedCmd == nil {
		return ""
	}
	cmd := m.selectedCmd

	var lines []string
	lines = append(lines, "")
	lines = append(lines, TitleStyle.Render("  Command Details"))
	lines = append(lines, "")
	lines = append(lines, "  "+CommandStyle.Render(cmd.Raw))
	lines = append(lines, "")
	lines = append(lines, MetadataStyle.Render(fmt.Sprintf("  Binary:     %s", cmd.Binary)))
	if cmd.Subcommand != "" {
		lines = append(lines, MetadataStyle.Render(fmt.Sprintf("  Subcommand: %s", cmd.Subcommand)))
	}
	lines = append(lines, MetadataStyle.Render(fmt.Sprintf("  Category:   %s", CategoryStyle.Render(cmd.Category))))
	lines = append(lines, MetadataStyle.Render(fmt.Sprintf("  Frequency:  %d", cmd.Frequency)))
	lines = append(lines, MetadataStyle.Render(fmt.Sprintf("  First seen: %s", cmd.FirstSeen.Format("2006-01-02 15:04"))))
	lines = append(lines, MetadataStyle.Render(fmt.Sprintf("  Last seen:  %s (%s)", cmd.LastSeen.Format("2006-01-02 15:04"), relativeTime(cmd.LastSeen))))
	if cmd.LastExit != nil {
		exitStyle := SuccessStyle
		if *cmd.LastExit != 0 {
			exitStyle = ErrorStyle
		}
		lines = append(lines, MetadataStyle.Render("  Last exit:  ")+exitStyle.Render(fmt.Sprintf("%d", *cmd.LastExit)))
	}

	if len(m.selectedCtxs) > 0 {
		lines = append(lines, "")
		lines = append(lines, TitleStyle.Render("  Recent Contexts"))
		for i, ctx := range m.selectedCtxs {
			if i >= 5 {
				lines = append(lines, DimStyle.Render(fmt.Sprintf("  ... and %d more", len(m.selectedCtxs)-5)))
				break
			}
			loc := ctx.Cwd
			if ctx.GitRepo != "" {
				loc = ctx.GitRepo
				if ctx.GitBranch != "" {
					loc += ":" + ctx.GitBranch
				}
			}
			lines = append(lines, DimStyle.Render(fmt.Sprintf("    %s  %s", ctx.Timestamp.Format("01/02 15:04"), loc)))
		}
	}

	lines = append(lines, "")
	lines = append(lines, HintStyle.Render("  [Esc] Back"))

	return strings.Join(lines, "\n")
}

func (m VaultBrowserModel) renderHelp() string {
	var lines []string
	lines = append(lines, "")
	lines = append(lines, TitleStyle.Render("  Keybindings"))
	lines = append(lines, "")

	bindings := [][2]string{
		{"/", "Search / filter commands"},
		{"Enter", "Select (categories) / Details (list)"},
		{"Tab", "Switch between Commands and Categories"},
		{"i", "View command details"},
		{"d", "Delete selected command"},
		{"s", "Cycle sort: recency → frequency → alpha"},
		{"j/k or ↑/↓", "Navigate up/down"},
		{"PgUp/PgDn", "Page up/down"},
		{"g/G", "Jump to top/bottom"},
		{"Backspace", "Clear category filter"},
		{"?", "Toggle this help"},
		{"q/Esc", "Quit"},
	}

	for _, b := range bindings {
		key := HintStyle.Render(fmt.Sprintf("  %-16s", b[0]))
		desc := NormalItemStyle.Render(b[1])
		lines = append(lines, key+desc)
	}

	lines = append(lines, "")
	lines = append(lines, HintStyle.Render("  [Esc] Back"))

	return strings.Join(lines, "\n")
}

func (m VaultBrowserModel) renderStatusBar() string {
	left := ""
	if m.confirmDelete {
		left = ErrorStyle.Render(" Delete this command? [y] Yes  [n] No")
	} else if m.searching {
		left = HintStyle.Render(" [Esc] Cancel search  [Enter] Apply")
	} else {
		left = HintStyle.Render(" [/] Search  [Tab] Switch view  [s] Sort  [i] Details  [d] Delete  [?] Help  [q] Quit")
	}

	pos := ""
	switch m.view {
	case viewList:
		if len(m.filtered) > 0 {
			pos = DimStyle.Render(fmt.Sprintf(" %d/%d ", m.cursor+1, len(m.filtered)))
		}
	case viewCategories:
		if len(m.categories) > 0 {
			pos = DimStyle.Render(fmt.Sprintf(" %d/%d ", m.cursor+1, len(m.categories)))
		}
	}

	maxLeft := m.width - lipgloss.Width(pos) - 2
	if maxLeft < 0 {
		maxLeft = 0
	}
	if lipgloss.Width(left) > maxLeft {
		left = ansi.Truncate(left, maxLeft, "")
	}

	return left + strings.Repeat(" ", max(0, m.width-lipgloss.Width(left)-lipgloss.Width(pos))) + pos
}

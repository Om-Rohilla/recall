package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	appctx "github.com/Om-Rohilla/recall/internal/context"
	"github.com/Om-Rohilla/recall/internal/explain"
	"github.com/Om-Rohilla/recall/internal/intelligence"
	"github.com/Om-Rohilla/recall/internal/ui"
	"github.com/Om-Rohilla/recall/internal/vault"
	"github.com/Om-Rohilla/recall/pkg/config"
	recallerr "github.com/Om-Rohilla/recall/pkg/errors"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"
)

var (
	searchTop       int
	searchNoExec    bool
	searchJSON      bool
	searchVaultOnly bool
	searchKBOnly    bool
	searchCategory  string
)

var searchCmd = &cobra.Command{
	Use:     "search [query]",
	Aliases: []string{"s"},
	Short:   "Search commands by intent",
	Long: `Search your command vault using natural language.

Examples:
  recall search "find large files"
  recall search "docker cleanup" --top 5
  recall s "undo git commit"`,
	Args: cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return executeSearch(args)
	},
}

func init() {
	searchCmd.Flags().IntVar(&searchTop, "top", 0, "show top N results")
	searchCmd.Flags().BoolVar(&searchNoExec, "no-execute", false, "show result without execute option")
	searchCmd.Flags().BoolVar(&searchJSON, "json", false, "output as JSON")
	searchCmd.Flags().BoolVar(&searchVaultOnly, "vault-only", false, "search only personal history")
	searchCmd.Flags().BoolVar(&searchKBOnly, "kb-only", false, "search only knowledge base")
	searchCmd.Flags().StringVar(&searchCategory, "category", "", "filter by category")
	rootCmd.AddCommand(searchCmd)
}

func runSearch(args []string) error {
	return executeSearch(args)
}

func executeSearch(args []string) error {
	query := strings.Join(args, " ")
	if query == "" {
		return fmt.Errorf("please provide a search query")
	}

	cfg := config.Get()
	topN := cfg.Search.TopResults
	if searchTop > 0 {
		topN = searchTop
	}
	// For interactive mode, fetch more results to navigate
	if !searchJSON && !searchNoExec && topN < 5 {
		topN = 5
	}

	store, err := vault.NewStore(cfg.Vault.Path)
	if err != nil {
		return recallerr.Vault(err)
	}
	defer store.Close()

	cwd, _ := os.Getwd()
	currentCtx := appctx.Detect(cwd)

	kbPath := intelligence.FindKnowledgeBasePath()

	opts := intelligence.SearchOptions{
		Limit:     topN,
		VaultOnly: searchVaultOnly,
		KBOnly:    searchKBOnly,
		Category:  searchCategory,
		KBPath:    kbPath,
	}

	engine := intelligence.NewEngine(store)
	results, err := engine.Search(query, currentCtx, opts)
	if err != nil {
		return fmt.Errorf("searching: %w", err)
	}

	// Filter by minimum confidence
	minConf := cfg.Search.MinConfidence * 100
	var filtered []vault.SearchResult
	for _, r := range results {
		if r.Confidence >= minConf {
			filtered = append(filtered, r)
		}
	}
	results = filtered

	if len(results) == 0 {
		fmt.Println(ui.RenderNoResults(query))
		return nil
	}

	// JSON mode: output and exit
	if searchJSON {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(results)
	}

	// Non-interactive mode: just print
	if searchNoExec {
		fmt.Println(ui.RenderResultList(results))
		return nil
	}

	// Interactive mode: launch Bubbletea picker
	return runInteractiveSearch(results)
}

// runInteractiveSearch launches the interactive result picker and handles the chosen action.
func runInteractiveSearch(results []vault.SearchResult) error {
	model := ui.NewSearchInteractive(results)
	p := tea.NewProgram(model)

	finalModel, err := p.Run()
	if err != nil {
		return fmt.Errorf("running interactive search: %w", err)
	}

	m, ok := finalModel.(ui.SearchInteractiveModel)
	if !ok {
		return nil
	}

	outcome := m.Outcome()
	if outcome == nil {
		return nil
	}

	cmdRaw := outcome.Command.Command.Raw

	switch outcome.Action {
	case ui.ActionExecute:
		// Print the command being executed
		fmt.Println()
		fmt.Println(ui.HintStyle.Render("  ▸ Executing: ") + ui.CommandStyle.Render(cmdRaw))
		fmt.Println()
		return ui.ExecuteCommand(cmdRaw)

	case ui.ActionCopy:
		// Already handled in the TUI, but as fallback:
		fmt.Println(ui.SuccessStyle.Render("  ✓ Copied: ") + ui.CommandStyle.Render(cmdRaw))
		return nil

	case ui.ActionEdit:
		edited, err := ui.EditCommand(cmdRaw)
		if err != nil {
			return fmt.Errorf("editing command: %w", err)
		}
		if edited == "" || edited == cmdRaw {
			fmt.Println(ui.MetadataStyle.Render("  No changes made."))
			return nil
		}
		fmt.Println()
		fmt.Println(ui.HintStyle.Render("  ▸ Executing edited command: ") + ui.CommandStyle.Render(edited))
		fmt.Println()
		return ui.ExecuteCommand(edited)

	case ui.ActionExplain:
		result := explain.Explain(cmdRaw)
		fmt.Println()
		fmt.Println(ui.RenderFullExplain(result, true))
		return nil
	}

	return nil
}

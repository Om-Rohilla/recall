package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	appctx "github.com/Om-Rohilla/recall/internal/context"
	"github.com/Om-Rohilla/recall/internal/intelligence"
	"github.com/Om-Rohilla/recall/internal/ui"
	"github.com/Om-Rohilla/recall/internal/vault"
	"github.com/Om-Rohilla/recall/pkg/config"
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

	store, err := vault.NewStore(cfg.Vault.Path)
	if err != nil {
		return fmt.Errorf("opening vault: %w", err)
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

	results, err := intelligence.Search(store, query, currentCtx, opts)
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

	if searchJSON {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(results)
	}

	fmt.Println(ui.RenderResultList(results))

	return nil
}

package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

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
	searchCmd.Flags().StringVar(&searchCategory, "category", "", "filter by category")
	rootCmd.AddCommand(searchCmd)
}

// runSearch is called when the first arg isn't a known subcommand.
// This replaces the stub in root.go.
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

	results, err := store.SearchFTS5(query, topN+10) // fetch extra for filtering
	if err != nil {
		return fmt.Errorf("searching vault: %w", err)
	}

	// Filter by category if specified
	if searchCategory != "" {
		var filtered []vault.SearchResult
		for _, r := range results {
			if strings.EqualFold(r.Command.Category, searchCategory) {
				filtered = append(filtered, r)
			}
		}
		results = filtered
	}

	// Filter by minimum confidence
	var filtered []vault.SearchResult
	for _, r := range results {
		if r.Confidence >= cfg.Search.MinConfidence*100 {
			filtered = append(filtered, r)
		}
	}
	results = filtered

	// Limit to topN
	if len(results) > topN {
		results = results[:topN]
	}

	if len(results) == 0 {
		fmt.Println(ui.RenderNoResults(query))
		os.Exit(2)
	}

	if searchJSON {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(results)
	}

	fmt.Println(ui.RenderResultList(results))

	return nil
}

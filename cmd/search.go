package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

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
		fmt.Println("No results found.")
		fmt.Println()
		fmt.Println("💡 Tips:")
		fmt.Println("   - Try different keywords")
		fmt.Println("   - Import your history: recall import-history")
		fmt.Println("   - Use broader terms")
		os.Exit(2)
	}

	if searchJSON {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(results)
	}

	// Render result cards (will use ui package in next step)
	for i, r := range results {
		if i > 0 {
			fmt.Println()
		}
		renderBasicResult(r, i == 0)
	}

	return nil
}

// renderBasicResult outputs a simple formatted result.
// Replaced by Lipgloss rendering in the UI step.
func renderBasicResult(r vault.SearchResult, isBest bool) {
	label := "Match"
	if isBest {
		label = "Best Match"
	}
	fmt.Printf("── %s (confidence: %.0f%%) ──\n", label, r.Confidence)
	fmt.Printf("  %s\n", r.Command.Raw)
	fmt.Println()
	if r.Command.Frequency > 1 {
		fmt.Printf("  Used %d times", r.Command.Frequency)
	}
	if !r.Command.LastSeen.IsZero() {
		fmt.Printf("  |  Last used: %s", r.Command.LastSeen.Format("2006-01-02"))
	}
	if r.Command.Category != "" {
		fmt.Printf("  |  Category: %s", r.Command.Category)
	}
	fmt.Println()
}

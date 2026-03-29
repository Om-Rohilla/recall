package cmd

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"

	"github.com/Om-Rohilla/recall/internal/ui"
	"github.com/Om-Rohilla/recall/internal/vault"
	"github.com/Om-Rohilla/recall/pkg/config"
)

var (
	vaultCategory string
	vaultProject  string
	vaultSort     string
)

var vaultCmd = &cobra.Command{
	Use:     "vault",
	Aliases: []string{"v"},
	Short:   "Browse your command vault (TUI)",
	Long: `Open a full terminal UI to browse, search, and manage your stored commands.

Features:
  - Full-text search with live filtering
  - Browse by category (git, docker, filesystem, network, system, etc.)
  - Sort by frequency, recency, or alphabetically
  - View command details, delete entries

Keybindings:
  /        Search/filter
  Tab      Switch views (commands/categories)
  s        Cycle sort mode
  i        View details
  d        Delete entry
  ?        Show help
  q/Esc    Quit`,
	RunE: runVault,
}

func init() {
	vaultCmd.Flags().StringVar(&vaultCategory, "category", "", "filter to specific category")
	vaultCmd.Flags().StringVar(&vaultProject, "project", "", "filter to specific project")
	vaultCmd.Flags().StringVar(&vaultSort, "sort", "recency", "sort by: frequency, recency, alpha")
	rootCmd.AddCommand(vaultCmd)
}

func runVault(cmd *cobra.Command, args []string) error {
	cfg := config.Get()

	store, err := vault.NewStore(cfg.Vault.Path)
	if err != nil {
		return fmt.Errorf("opening vault: %w", err)
	}
	defer store.Close()

	model := ui.NewVaultBrowser(store, vaultCategory, vaultSort)
	p := tea.NewProgram(model, tea.WithAltScreen())

	if _, err := p.Run(); err != nil {
		return fmt.Errorf("running vault browser: %w", err)
	}

	return nil
}

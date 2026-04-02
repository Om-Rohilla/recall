package cmd

import (
	"fmt"
	"html"
	"os"
	"path/filepath"
	"strings"

	"github.com/Om-Rohilla/recall/internal/ui"
	"github.com/Om-Rohilla/recall/internal/vault"
	"github.com/Om-Rohilla/recall/pkg/config"
	"github.com/spf13/cobra"
)

var shareCmd = &cobra.Command{
	Use:   "share [command...]",
	Short: "Export a beautiful SVG card of a command for social sharing",
	RunE:  runShare,
}

func init() {
	rootCmd.AddCommand(shareCmd)
}

func runShare(cmd *cobra.Command, args []string) error {
	rawQuery := strings.Join(args, " ")
	if rawQuery == "" {
		return fmt.Errorf("please provide a command to share")
	}

	cfg := config.Get()
	store, err := vault.NewStore(cfg.Vault.Path)
	if err != nil {
		return fmt.Errorf("opening vault: %w", err)
	}
	defer store.Close()

	// Try to find the actual command in history for stats
	results, err := store.SearchFTS5(rawQuery, 1)
	if err != nil {
		return fmt.Errorf("searching vault: %w", err)
	}

	title := "recall"
	actualCmd := rawQuery
	footer := "Powered by Recall CLI"

	if len(results) > 0 {
		actualCmd = results[0].Command.Raw
		footer = fmt.Sprintf("Used %d times  •  Powered by Recall", results[0].Command.Frequency)
	}

	svg := generateSVG(title, actualCmd, footer)
	
	outPath := "recall_share.svg"
	if err := os.WriteFile(outPath, []byte(svg), 0644); err != nil {
		return fmt.Errorf("writing SVG: %w", err)
	}

	pwd, _ := os.Getwd()
	fullPath := filepath.Join(pwd, outPath)
	fmt.Println(ui.SuccessStyle.Render(fmt.Sprintf("✨ Beautiful share card generated at: %s", fullPath)))
	return nil
}

func generateSVG(title, content, footer string) string {
	content = html.EscapeString(content)
	title = html.EscapeString(title)
	footer = html.EscapeString(footer)

	// Modern macOS Terminal-like aesthetic
	return fmt.Sprintf(`<?xml version="1.0" encoding="utf-8"?>
<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 800 240" width="800" height="240">
  <defs>
    <linearGradient id="bg" x1="0%%" y1="0%%" x2="100%%" y2="100%%">
      <stop offset="0%%" stop-color="#282a36" />
      <stop offset="100%%" stop-color="#1e1f29" />
    </linearGradient>
    <linearGradient id="primary" x1="0%%" y1="0%%" x2="100%%" y2="0%%">
      <stop offset="0%%" stop-color="#ff79c6" />
      <stop offset="100%%" stop-color="#bd93f9" />
    </linearGradient>
  </defs>

  <!-- Background with rounded corners and shadow -->
  <rect x="20" y="20" width="760" height="200" rx="12" fill="url(#bg)" stroke="#44475a" stroke-width="2"/>

  <!-- macOS Window Controls -->
  <circle cx="50" cy="50" r="6" fill="#ff5f56" />
  <circle cx="70" cy="50" r="6" fill="#ffbd2e" />
  <circle cx="90" cy="50" r="6" fill="#27c93f" />

  <!-- Window Title -->
  <text x="400" y="55" font-family="-apple-system, system-ui, sans-serif" font-size="14" fill="#6272a4" text-anchor="middle" font-weight="600">%s</text>

  <!-- Command Content -->
  <text x="50" y="110" font-family="'Fira Code', 'Courier New', monospace" font-size="24" fill="#f8f8f2" font-weight="bold">> <tspan fill="url(#primary)">%s</tspan></text>

  <!-- Footer Stats -->
  <text x="380" y="180" font-family="-apple-system, system-ui, sans-serif" font-size="13" fill="#6272a4" text-anchor="end">%s</text>
  <text x="740" y="180" font-family="-apple-system, system-ui, sans-serif" font-size="13" fill="#6272a4" text-anchor="end">recall.sh</text>
</svg>`, title, content, footer)
}

package cmd

import (
	"fmt"
	"strings"

	"github.com/Om-Rohilla/recall/internal/vault"
	"github.com/Om-Rohilla/recall/pkg/config"
	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"
)

var roastCmd = &cobra.Command{
	Use:   "roast",
	Short: "Get roasted for your terrible command line habits",
	RunE:  runRoast,
}

func init() {
	rootCmd.AddCommand(roastCmd)
}

func runRoast(cmd *cobra.Command, args []string) error {
	cfg := config.Get()

	store, err := vault.NewStore(cfg.Vault.Path)
	if err != nil {
		return fmt.Errorf("opening vault: %w", err)
	}
	defer store.Close()

	commands, err := store.GetRecentCommands(1000)
	if err != nil {
		return fmt.Errorf("fetching history: %w", err)
	}

	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#FF5555")). // Red
		MarginBottom(1).
		Render("🔥 TERMINAL ROAST 🔥")

	boxStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#FF5555")).
		Padding(1, 3).
		MarginTop(1).
		MarginBottom(1)

	var roasts []string

	// Simple heuristic roasting
	chmod777 := 0
	rmRf := 0
	forcePushes := 0
	sudoAbuse := 0
	crying := 0

	for _, c := range commands {
		raw := strings.ToLower(c.Raw)
		if strings.Contains(raw, "chmod 777") || strings.Contains(raw, "chmod -r 777") || strings.Contains(raw, "chmod -R 777") {
			chmod777++
		}
		if strings.Contains(raw, "rm -rf") || strings.Contains(raw, "rm -fr") {
			rmRf++
		}
		if strings.Contains(raw, "git push -f") || strings.Contains(raw, "git push --force") {
			forcePushes++
		}
		if strings.HasPrefix(raw, "sudo ") {
			sudoAbuse++
		}
		if strings.Contains(raw, "exit") || strings.Contains(raw, "clear") {
             crying++
        }
	}

	if chmod777 > 0 {
		roasts = append(roasts, fmt.Sprintf("• %d times you used 'chmod 777'. Who hurt you? Security is not optional.", chmod777))
	}
	if forcePushes > 0 {
		roasts = append(roasts, fmt.Sprintf("• %d force pushes... Your coworkers definitely have a Slack channel without you.", forcePushes))
	}
	if rmRf > 0 {
		roasts = append(roasts, fmt.Sprintf("• %d times typing 'rm -rf'. I'm taking away your keyboard privileges.", rmRf))
	}
	if sudoAbuse > 50 {
		roasts = append(roasts, "• You hit 'sudo' like a panic button. Learn file permissions!")
	}

	var sb strings.Builder
	sb.WriteString(titleStyle + "\n")

	if len(roasts) == 0 {
		sb.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("#50fa7b")).Render("Surprisingly, your recent history is... passable.\nBoring, but passable. Go break production and come back."))
	} else {
		sb.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("#f1fa8c")).Render("I analyzed your recent history. I am deeply disappointed.\n\n"))
		for _, r := range roasts {
			sb.WriteString(r + "\n")
		}
	}

	fmt.Println(boxStyle.Render(sb.String()))
	return nil
}

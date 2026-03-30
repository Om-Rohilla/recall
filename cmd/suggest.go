package cmd

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/Om-Rohilla/recall/internal/ui"
	"github.com/Om-Rohilla/recall/internal/vault"
	"github.com/Om-Rohilla/recall/pkg/config"
)

var (
	suggestMinFreq int
	suggestDryRun  bool
	suggestShell   string
)

var suggestCmd = &cobra.Command{
	Use:     "suggest-aliases",
	Aliases: []string{"sa"},
	Short:   "Suggest shell aliases for repetitive commands",
	Long: `Analyze your command frequency patterns and suggest shell aliases
for commands you type frequently.

Examples:
  recall suggest-aliases
  recall sa --min-freq 10
  recall sa --dry-run`,
	RunE: runSuggest,
}

func init() {
	suggestCmd.Flags().IntVar(&suggestMinFreq, "min-freq", 0, "minimum frequency to suggest (default: from config)")
	suggestCmd.Flags().BoolVar(&suggestDryRun, "dry-run", false, "show suggestions without writing to shell config")
	suggestCmd.Flags().StringVar(&suggestShell, "shell", "", "target shell (auto-detected)")
	rootCmd.AddCommand(suggestCmd)
}

func runSuggest(cmd *cobra.Command, args []string) error {
	cfg := config.Get()

	store, err := vault.NewStore(cfg.Vault.Path)
	if err != nil {
		return fmt.Errorf("opening vault: %w", err)
	}
	defer store.Close()

	minFreq := cfg.Alias.MinFrequency
	if suggestMinFreq > 0 {
		minFreq = suggestMinFreq
	}

	cmds, err := store.GetHighFrequencyCommands(minFreq)
	if err != nil {
		return fmt.Errorf("getting high frequency commands: %w", err)
	}

	suggestions := generateAliases(cmds)

	if len(suggestions) == 0 {
		fmt.Println(ui.BorderStyle.Render(
			ui.TitleStyle.Render("No Alias Suggestions") + "\n\n" +
				ui.MetadataStyle.Render(fmt.Sprintf("  No commands found with frequency >= %d.\n  Keep using your terminal and check back later!", minFreq)),
		))
		return nil
	}

	uiSuggestions := make([]ui.AliasSuggestion, len(suggestions))
	for i, s := range suggestions {
		uiSuggestions[i] = ui.AliasSuggestion{Command: s.Command, Alias: s.Alias, Frequency: s.Frequency}
	}
	fmt.Println(ui.RenderAliasSuggestions(uiSuggestions))
	return nil
}

type aliasSuggestion struct {
	Command   string
	Alias     string
	Frequency int
}

func generateAliases(cmds []vault.Command) []aliasSuggestion {
	var suggestions []aliasSuggestion

	for _, cmd := range cmds {
		raw := strings.TrimSpace(cmd.Raw)
		if len(raw) <= 5 {
			continue
		}
		if strings.Contains(raw, "recall ") {
			continue
		}

		alias := generateAliasName(raw, cmd.Binary, cmd.Subcommand)
		if alias == "" || alias == cmd.Binary {
			continue
		}
		if len(alias) >= len(raw) {
			continue
		}

		suggestions = append(suggestions, aliasSuggestion{
			Command:   raw,
			Alias:     alias,
			Frequency: cmd.Frequency,
		})
	}

	seen := make(map[string]bool)
	var unique []aliasSuggestion
	for _, s := range suggestions {
		if !seen[s.Alias] {
			seen[s.Alias] = true
			unique = append(unique, s)
		}
	}

	maxSuggestions := 10
	if len(unique) > maxSuggestions {
		unique = unique[:maxSuggestions]
	}
	return unique
}

var knownAliases = map[string]string{
	"git status":                      "gst",
	"git add":                         "ga",
	"git add -A":                      "gaa",
	"git commit":                      "gc",
	"git push":                        "gp",
	"git pull":                        "gpl",
	"git pull --rebase":               "gpr",
	"git checkout":                    "gco",
	"git branch":                      "gb",
	"git log --oneline":               "glog",
	"git log --oneline --graph":       "glg",
	"git log --oneline --graph --all": "glga",
	"git diff":                        "gd",
	"git diff --cached":               "gdc",
	"git stash":                       "gsta",
	"git stash pop":                   "gstp",
	"git merge":                       "gm",
	"git rebase":                      "grb",
	"docker ps":                       "dps",
	"docker ps -a":                    "dpsa",
	"docker images":                   "dim",
	"docker compose up":               "dcu",
	"docker compose up -d":            "dcud",
	"docker compose down":             "dcd",
	"docker compose logs":             "dcl",
	"docker system prune":             "dsp",
	"kubectl get pods":                "kgp",
	"kubectl get services":            "kgs",
	"kubectl get deployments":         "kgd",
	"kubectl get namespaces":          "kgns",
	"kubectl describe":                "kd",
	"kubectl logs":                    "kl",
	"kubectl apply -f":                "kaf",
	"npm install":                     "ni",
	"npm run":                         "nr",
	"npm test":                        "nt",
	"go test ./...":                   "gt",
	"go build":                        "gob",
	"go run":                          "gor",
	"python3":                         "py",
	"python":                          "py",
}

func generateAliasName(raw, binary, subcommand string) string {
	if binary == "" {
		return ""
	}

	bestAlias := ""
	bestLen := 0
	for prefix, alias := range knownAliases {
		if raw == prefix || strings.HasPrefix(raw, prefix+" ") {
			if len(prefix) > bestLen {
				bestLen = len(prefix)
				bestAlias = alias
			}
		}
	}
	if bestAlias != "" {
		return bestAlias
	}

	parts := strings.Fields(raw)
	if len(parts) < 2 {
		return ""
	}

	if subcommand != "" {
		subParts := strings.Fields(subcommand)
		alias := string(binary[0])
		for _, sp := range subParts {
			if len(sp) > 0 {
				alias += string(sp[0])
			}
		}
		if len(alias) >= 2 {
			return alias
		}
	}

	if len(parts) >= 2 && strings.HasPrefix(parts[1], "-") {
		flag := strings.TrimLeft(parts[1], "-")
		if len(flag) > 0 {
			return string(binary[0]) + string(flag[0])
		}
	}

	return ""
}

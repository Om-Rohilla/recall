package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/atotto/clipboard"

	"github.com/Om-Rohilla/recall/internal/compose"
	"github.com/Om-Rohilla/recall/internal/explain"
	"github.com/Om-Rohilla/recall/internal/ui"
	"github.com/Om-Rohilla/recall/internal/vault"
	"github.com/Om-Rohilla/recall/pkg/config"
	"github.com/spf13/cobra"
)

var composeCmd = &cobra.Command{
	Use:     "compose",
	Aliases: []string{"c"},
	Short:   "Build a command interactively step-by-step",
	Long: `Compose builds a command through an interactive wizard.
Choose a tool, select options step by step, and get the final command.

Supports: find, tar, grep, docker, git, kubectl, ssh, curl, rsync, chmod, and more.

Example:
  recall compose`,
	RunE: runCompose,
}

func init() {
	rootCmd.AddCommand(composeCmd)
}

type composeStyles struct{}

func (composeStyles) Title(s string) string    { return ui.TitleStyle.Render(s) }
func (composeStyles) Metadata(s string) string { return ui.MetadataStyle.Render(s) }
func (composeStyles) Command(s string) string  { return ui.CommandStyle.Render(s) }
func (composeStyles) Hint(s string) string     { return ui.HintStyle.Render(s) }
func (composeStyles) Error(s string) string    { return ui.ErrorStyle.Render(s) }
func (composeStyles) Warning(s string) string  { return ui.WarningStyle.Render(s) }

func runCompose(cmd *cobra.Command, args []string) error {
	reader := bufio.NewReader(os.Stdin)
	p := compose.New(reader, composeStyles{})

	fmt.Println(ui.TitleStyle.Render("Recall Command Composer"))
	fmt.Println()

	tool := p.PromptInput("What tool do you want to use?", "")
	if tool == "" {
		return fmt.Errorf("no tool specified")
	}

	toolInfo := explain.GetToolInfo(tool)
	if toolInfo != nil {
		fmt.Println(ui.MetadataStyle.Render(fmt.Sprintf("  %s — %s", tool, toolInfo.Description)))
		fmt.Println()
	}

	composed := p.Route(tool)
	if composed == "" {
		return nil
	}

	fmt.Println()
	fmt.Println(ui.TitleStyle.Render("Generated command:"))
	fmt.Println()
	fmt.Println("  " + ui.CommandStyle.Render(composed))
	fmt.Println()

	result := explain.Explain(composed)
	if result.DangerLevel == explain.Destructive {
		fmt.Println(ui.ErrorStyle.Render("  [!] This command has DESTRUCTIVE flags — review carefully"))
	} else if result.DangerLevel == explain.Caution {
		fmt.Println(ui.WarningStyle.Render("  [~] This command has flags that require caution"))
	}
	for _, w := range result.Warnings {
		if w.Level == explain.Destructive {
			fmt.Println(ui.ErrorStyle.Render("  [!] " + w.Message))
		}
	}

	cfg := config.Get()
	store, err := vault.NewStore(cfg.Vault.Path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: could not save to vault: %v\n", err)
	} else {
		defer store.Close()
		if saveErr := compose.SaveComposedCommand(store, composed); saveErr != nil {
			fmt.Fprintf(os.Stderr, "Warning: could not save composed command: %v\n", saveErr)
		}
	}

	// Interactive post-action menu
	for {
		fmt.Println()
		fmt.Println(ui.HintStyle.Render("  [Enter] Execute  [c] Copy  [e] Edit  [x] Explain  [q] Quit"))
		fmt.Print("  > ")

		input, _ := reader.ReadString('\n')
		input = strings.TrimSpace(strings.ToLower(input))

		switch input {
		case "", "enter":
			// Execute the composed command
			fmt.Println()
			fmt.Println(ui.HintStyle.Render("  ▸ Executing: ") + ui.CommandStyle.Render(composed))
			fmt.Println()
			return ui.ExecuteCommand(composed)

		case "c":
			if err := clipboard.WriteAll(composed); err != nil {
				fmt.Println(ui.ErrorStyle.Render("  ✗ Could not copy to clipboard: " + err.Error()))
			} else {
				fmt.Println(ui.SuccessStyle.Render("  ✓ Copied to clipboard!"))
			}
			continue

		case "e":
			edited, editErr := ui.EditCommand(composed)
			if editErr != nil {
				fmt.Println(ui.ErrorStyle.Render("  ✗ Error opening editor: " + editErr.Error()))
				continue
			}
			if edited != "" && edited != composed {
				composed = edited
				fmt.Println()
				fmt.Println(ui.TitleStyle.Render("  Updated command:"))
				fmt.Println("  " + ui.CommandStyle.Render(composed))
			} else {
				fmt.Println(ui.MetadataStyle.Render("  No changes made."))
			}
			continue

		case "x":
			explainResult := explain.Explain(composed)
			fmt.Println()
			fmt.Println(ui.RenderFullExplain(explainResult, true))
			continue

		case "q":
			return nil

		default:
			fmt.Println(ui.MetadataStyle.Render("  Unknown action. Use c/e/x/q or press Enter to execute."))
			continue
		}
	}
}

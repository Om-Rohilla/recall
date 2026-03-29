package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/Om-Rohilla/recall/internal/explain"
	"github.com/Om-Rohilla/recall/internal/ui"
	"github.com/spf13/cobra"
)

var (
	explainShort      bool
	explainJSON       bool
	explainNoWarnings bool
)

var explainCmd = &cobra.Command{
	Use:     "explain [command]",
	Aliases: []string{"e"},
	Short:   "Break down a command into plain English",
	Long: `Explain any command by decomposing it into annotated components.

Each flag, argument, and operator is explained in plain English.
Destructive flags are highlighted with warnings.

Examples:
  recall explain "find . -name '*.log' -mtime +30 -delete"
  recall explain "tar -xzvf archive.tar.gz -C /opt/"
  recall explain "rsync -avz --delete ./src/ user@host:/deploy/"
  recall e "chmod -R 755 /var/www"`,
	Args: cobra.MinimumNArgs(1),
	RunE: runExplain,
}

func init() {
	explainCmd.Flags().BoolVar(&explainShort, "short", false, "one-line summary only")
	explainCmd.Flags().BoolVar(&explainJSON, "json", false, "output as JSON")
	explainCmd.Flags().BoolVar(&explainNoWarnings, "no-warnings", false, "skip safety warnings")
	rootCmd.AddCommand(explainCmd)
}

func runExplain(cmd *cobra.Command, args []string) error {
	raw := strings.Join(args, " ")
	if raw == "" {
		return fmt.Errorf("please provide a command to explain")
	}

	result := explain.Explain(raw)

	if explainJSON {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(result)
	}

	if explainShort {
		fmt.Println(renderShortExplain(result))
		return nil
	}

	fmt.Println(renderFullExplain(result, !explainNoWarnings))
	return nil
}

func renderShortExplain(result explain.ExplainResult) string {
	parts := []string{result.Binary}
	for _, c := range result.Components {
		if c.Type == "subcommand" {
			parts = append(parts, c.Token)
		}
	}
	summary := strings.Join(parts, " ")
	if result.Summary != "" {
		summary = result.Summary
	}

	dangerIcon := ""
	switch result.DangerLevel {
	case explain.Destructive:
		dangerIcon = ui.ErrorStyle.Render(" [DESTRUCTIVE]")
	case explain.Caution:
		dangerIcon = ui.WarningStyle.Render(" [CAUTION]")
	}

	return ui.CommandStyle.Render(result.Raw) + " — " + ui.MetadataStyle.Render(summary) + dangerIcon
}

func renderFullExplain(result explain.ExplainResult, showWarnings bool) string {
	var lines []string

	// Title
	titleLabel := "Command Breakdown"
	switch result.DangerLevel {
	case explain.Destructive:
		titleLabel = ui.ErrorStyle.Render("Command Breakdown [DESTRUCTIVE]")
	case explain.Caution:
		titleLabel = ui.WarningStyle.Render("Command Breakdown [CAUTION]")
	default:
		titleLabel = ui.TitleStyle.Render("Command Breakdown")
	}
	lines = append(lines, titleLabel)
	lines = append(lines, "")

	// Render each component with alignment
	maxTokenLen := 0
	for _, c := range result.Components {
		if len(c.Token) > maxTokenLen {
			maxTokenLen = len(c.Token)
		}
	}
	if maxTokenLen > 30 {
		maxTokenLen = 30
	}

	prevIsBinary := false
	for _, c := range result.Components {
		token := c.Token
		if len(token) > 30 {
			token = token[:27] + "..."
		}

		indent := "  "
		if c.Type == "binary" && !prevIsBinary {
			indent = ""
		}

		var descStyled string
		switch c.Danger {
		case explain.Destructive:
			marker := ui.ErrorStyle.Render("[!] ")
			descStyled = marker + ui.ErrorStyle.Render(c.Description)
		case explain.Caution:
			marker := ui.WarningStyle.Render("[~] ")
			descStyled = marker + ui.WarningStyle.Render(c.Description)
		default:
			descStyled = ui.MetadataStyle.Render(c.Description)
		}

		padding := strings.Repeat(" ", max(1, maxTokenLen+2-len(token)))
		tokenStyled := ui.CommandStyle.Render(indent + token)

		lines = append(lines, tokenStyled+padding+ui.MetadataStyle.Render("<- ")+descStyled)

		prevIsBinary = c.Type == "binary"
	}

	// Warnings
	if showWarnings && len(result.Warnings) > 0 {
		lines = append(lines, "")
		for _, w := range result.Warnings {
			switch w.Level {
			case explain.Destructive:
				lines = append(lines, ui.ErrorStyle.Render("[!] "+w.Message))
			case explain.Caution:
				lines = append(lines, ui.WarningStyle.Render("[~] "+w.Message))
			}
		}
	}

	// Suggestions
	if showWarnings && len(result.Suggestions) > 0 {
		lines = append(lines, "")
		for _, s := range result.Suggestions {
			lines = append(lines, ui.HintStyle.Render("Tip: "+s))
		}
	}

	content := strings.Join(lines, "\n")
	return ui.BorderStyle.Render(content)
}

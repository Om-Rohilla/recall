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
		fmt.Println(ui.RenderShortExplain(result))
		return nil
	}

	fmt.Println(ui.RenderFullExplain(result, !explainNoWarnings))
	return nil
}

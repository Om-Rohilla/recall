package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/Om-Rohilla/recall/internal/capture"
	"github.com/Om-Rohilla/recall/internal/vault"
	"github.com/Om-Rohilla/recall/pkg/config"
	"github.com/Om-Rohilla/recall/pkg/shell"
	"github.com/spf13/cobra"
)

var (
	importFile  string
	importLimit int
	importDry   bool
)

var importHistoryCmd = &cobra.Command{
	Use:     "import-history",
	Aliases: []string{"ih"},
	Short:   "Import existing shell history into vault",
	Long: `Import your existing shell history file into the Recall vault.
This gives Recall a head start with your personal command patterns.

Supports:
  - Zsh history (with extended history timestamps)
  - Bash history
  - Fish history (YAML format)`,
	RunE: runImportHistory,
}

func init() {
	importHistoryCmd.Flags().StringVar(&importFile, "file", "", "import from specific history file")
	importHistoryCmd.Flags().IntVar(&importLimit, "limit", 0, "import only last N commands (0 = all)")
	importHistoryCmd.Flags().BoolVar(&importDry, "dry-run", false, "show what would be imported")
	rootCmd.AddCommand(importHistoryCmd)
}

func runImportHistory(cmd *cobra.Command, args []string) error {
	cfg := config.Get()

	// Determine history file path
	histPath := importFile
	if histPath == "" {
		shellInfo, err := shell.Detect()
		if err != nil {
			return fmt.Errorf("detecting shell: %w (use --file to specify history file)", err)
		}
		histPath = shellInfo.HistoryPath
	}

	fmt.Printf("📂 Reading history from: %s\n", histPath)

	f, err := os.Open(histPath)
	if err != nil {
		return fmt.Errorf("opening history file: %w", err)
	}
	defer f.Close()

	// Read all lines
	var lines []string
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 1024*1024), 1024*1024) // 1MB buffer for long lines
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	if err := scanner.Err(); err != nil {
		return fmt.Errorf("reading history: %w", err)
	}

	fmt.Printf("   Found %d raw lines\n", len(lines))

	// Apply limit (from end)
	if importLimit > 0 && importLimit < len(lines) {
		lines = lines[len(lines)-importLimit:]
	}

	// Parse lines into commands, joining multiline entries.
	// For Fish shell, use the YAML-format parser.
	var commands []vault.Command
	if shellInfo, err := shell.Detect(); err == nil && shellInfo.Shell == shell.Fish {
		commands = parseFishHistory(lines, cfg)
	} else {
		commands = parseHistoryLines(lines, cfg)
	}

	fmt.Printf("   Parsed %d unique commands (after filtering)\n", len(commands))

	if importDry {
		fmt.Println("\n📋 Dry run — commands that would be imported:")
		limit := len(commands)
		if limit > 20 {
			limit = 20
		}
		for i, cmd := range commands[:limit] {
			fmt.Printf("   %d. %s\n", i+1, truncate(cmd.Raw, 80))
		}
		if len(commands) > 20 {
			fmt.Printf("   ... and %d more\n", len(commands)-20)
		}
		return nil
	}

	// Open vault and batch insert
	store, err := vault.NewStore(cfg.Vault.Path)
	if err != nil {
		return fmt.Errorf("opening vault: %w", err)
	}
	defer store.Close()

	inserted, err := store.BatchInsertCommands(commands)
	if err != nil {
		return fmt.Errorf("importing commands: %w", err)
	}

	// Rebuild FTS index after bulk import
	if err := store.RebuildFTSIndex(); err != nil {
		fmt.Fprintf(os.Stderr, "warning: FTS index rebuild failed: %v\n", err)
	}

	fmt.Printf("\n✅ Imported %d commands into vault\n", inserted)
	fmt.Println("💡 Try: recall \"find files\"")

	return nil
}

func parseHistoryLines(lines []string, cfg *config.Config) []vault.Command {
	seen := make(map[string]bool)
	var commands []vault.Command
	var multiline strings.Builder

	for _, line := range lines {
		// Handle zsh multiline continuation
		if strings.HasSuffix(line, "\\") {
			multiline.WriteString(strings.TrimSuffix(line, "\\"))
			multiline.WriteString(" ")
			continue
		}

		if multiline.Len() > 0 {
			multiline.WriteString(line)
			line = multiline.String()
			multiline.Reset()
		}

		cmd := capture.ProcessHistoryLine(line, cfg)
		if cmd == nil {
			continue
		}

		if seen[cmd.Raw] {
			continue
		}
		seen[cmd.Raw] = true

		commands = append(commands, *cmd)
	}

	return commands
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}

// parseFishHistory parses Fish shell's YAML-based history format:
//
//	- cmd: git status
//	  when: 1609459200
//	- cmd: docker ps
//	  when: 1609459260
func parseFishHistory(lines []string, cfg *config.Config) []vault.Command {
	seen := make(map[string]bool)
	var commands []vault.Command
	var currentCmd string

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "- cmd:") {
			// Flush previous command before starting a new one
			if currentCmd != "" {
				if cmd := capture.ProcessHistoryLine(currentCmd, cfg); cmd != nil && !seen[cmd.Raw] {
					seen[cmd.Raw] = true
					commands = append(commands, *cmd)
				}
				currentCmd = ""
			}
			currentCmd = strings.TrimSpace(strings.TrimPrefix(line, "- cmd:"))
		} else if strings.HasPrefix(line, "cmd:") {
			currentCmd = strings.TrimSpace(strings.TrimPrefix(line, "cmd:"))
		} else if strings.HasPrefix(line, "when:") || line == "" {
			// Timestamp line or blank line signals end of current entry
			if currentCmd != "" {
				if cmd := capture.ProcessHistoryLine(currentCmd, cfg); cmd != nil && !seen[cmd.Raw] {
					seen[cmd.Raw] = true
					commands = append(commands, *cmd)
				}
				currentCmd = ""
			}
		}
	}

	// Flush any trailing command that wasn't followed by a "when:" line
	if currentCmd != "" {
		if cmd := capture.ProcessHistoryLine(currentCmd, cfg); cmd != nil && !seen[cmd.Raw] {
			commands = append(commands, *cmd)
		}
	}

	return commands
}

package capture

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/Om-Rohilla/recall/internal/vault"
	"github.com/Om-Rohilla/recall/pkg/config"
)

// ProcessCommand is the main entry point for the capture pipeline.
// It parses, filters, enriches, and stores a command.
func ProcessCommand(store *vault.Store, data *vault.CaptureData, cfg *config.Config) error {
	if !cfg.Capture.Enabled {
		return nil
	}

	// Step 1: Filter
	filterResult := Filter(data.RawCommand, cfg)
	if !filterResult.Allowed {
		return nil
	}

	// Step 2: Parse
	parsed := Parse(data.RawCommand)

	// Step 3: Enrich
	enrichment := Enrich(data.Cwd)
	if data.GitRepo == "" {
		data.GitRepo = enrichment.GitRepo
	}
	if data.GitBranch == "" {
		data.GitBranch = enrichment.GitBranch
	}
	if data.ProjectType == "" {
		data.ProjectType = enrichment.ProjectType
	}

	// Step 4: Store
	flagsJSON, _ := json.Marshal(parsed.Flags)
	exitCode := data.ExitCode

	cmd := &vault.Command{
		Raw:        parsed.Raw,
		Binary:     parsed.Binary,
		Subcommand: parsed.Subcommand,
		Flags:      string(flagsJSON),
		Category:   parsed.Category,
		Frequency:  1,
		FirstSeen:  data.Timestamp,
		LastSeen:   data.Timestamp,
		LastExit:   &exitCode,
	}

	if data.DurationMs > 0 {
		dur := float64(data.DurationMs)
		cmd.AvgDuration = &dur
	}

	cmdID, err := store.InsertCommand(cmd)
	if err != nil {
		return fmt.Errorf("storing command: %w", err)
	}

	// Store context
	ctx := &vault.Context{
		CommandID:   cmdID,
		Cwd:         data.Cwd,
		GitRepo:     data.GitRepo,
		GitBranch:   data.GitBranch,
		ProjectType: data.ProjectType,
		Timestamp:   data.Timestamp,
		ExitCode:    &exitCode,
		SessionID:   data.SessionID,
	}
	if data.DurationMs > 0 {
		ctx.DurationMs = &data.DurationMs
	}

	if err := store.InsertContext(ctx); err != nil {
		return fmt.Errorf("storing context: %w", err)
	}

	return nil
}

// ProcessHistoryLine parses a single line from shell history into a Command.
// Used by import-history. Returns nil if the line should be skipped.
func ProcessHistoryLine(line string, cfg *config.Config) *vault.Command {
	line = cleanHistoryLine(line)
	if line == "" {
		return nil
	}

	filterResult := Filter(line, cfg)
	if !filterResult.Allowed {
		return nil
	}

	parsed := Parse(line)
	if parsed.Binary == "" {
		return nil
	}

	flagsJSON, _ := json.Marshal(parsed.Flags)
	now := time.Now().UTC()

	return &vault.Command{
		Raw:        parsed.Raw,
		Binary:     parsed.Binary,
		Subcommand: parsed.Subcommand,
		Flags:      string(flagsJSON),
		Category:   parsed.Category,
		Frequency:  1,
		FirstSeen:  now,
		LastSeen:   now,
	}
}

// cleanHistoryLine strips shell history metadata from a line.
func cleanHistoryLine(line string) string {
	// Zsh extended history format: : 1234567890:0;actual command
	if len(line) > 0 && line[0] == ':' {
		if idx := findIndex(line, ';'); idx >= 0 {
			return line[idx+1:]
		}
	}
	return line
}

func findIndex(s string, b byte) int {
	for i := 0; i < len(s); i++ {
		if s[i] == b {
			return i
		}
	}
	return -1
}

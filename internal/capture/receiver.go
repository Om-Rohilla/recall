package capture

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/Om-Rohilla/recall/internal/vault"
	"github.com/Om-Rohilla/recall/pkg/config"
	"github.com/Om-Rohilla/recall/pkg/logging"
)

// minCaptureIntervalMs is the minimum interval between captures (in ms).
// Prevents flood attacks from malicious scripts.
const minCaptureIntervalMs = 50

// checkRateLimit uses a file-based timestamp to throttle captures.
// Returns true if the capture should proceed, false if rate limited.
func checkRateLimit(vaultPath string) bool {
	log := logging.Get()
	lockFile := filepath.Join(filepath.Dir(vaultPath), ".capture_ts")

	// Read last capture timestamp
	data, err := os.ReadFile(lockFile)
	if err == nil {
		if lastMs, err := strconv.ParseInt(strings.TrimSpace(string(data)), 10, 64); err == nil {
			elapsed := time.Now().UnixMilli() - lastMs
			if elapsed < minCaptureIntervalMs {
				log.Debug("capture rate limited", "elapsed_ms", elapsed)
				return false
			}
		}
	}

	// Write current timestamp
	nowMs := fmt.Sprintf("%d", time.Now().UnixMilli())
	_ = os.WriteFile(lockFile, []byte(nowMs), 0o600)
	return true
}

// ProcessCommand is the main entry point for the capture pipeline.
// It parses, filters, enriches, and stores a command.
func ProcessCommand(store *vault.Store, data *vault.CaptureData, cfg *config.Config) error {
	log := logging.Get()

	if !cfg.Capture.Enabled {
		log.Debug("capture disabled, skipping")
		return nil
	}

	// Input validation
	if data == nil {
		return fmt.Errorf("capture data is nil")
	}
	if len(data.RawCommand) > 10000 {
		log.Debug("command too long, skipping", "length", len(data.RawCommand))
		return nil
	}

	// Rate limiting — prevent flood attacks
	if !checkRateLimit(cfg.Vault.Path) {
		return nil
	}

	filterResult := Filter(data.RawCommand, cfg)
	if !filterResult.Allowed {
		log.Debug("command filtered", "reason", filterResult.Reason, "command", data.RawCommand)
		return nil
	}

	// Step 2: Parse
	parsed := Parse(data.RawCommand)

	// Step 3: Enrich (only if context data missing — avoids expensive git subprocess calls)
	if data.GitRepo == "" || data.GitBranch == "" || data.ProjectType == "" {
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
	}

	// Step 4: Store
	flagsJSON, marshalErr := json.Marshal(parsed.Flags)
	if marshalErr != nil {
		log.Warn("failed to marshal flags", "error", marshalErr)
		flagsJSON = []byte("[]")
	}
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

	histFlagsJSON, _ := json.Marshal(parsed.Flags)
	now := time.Now().UTC()

	return &vault.Command{
		Raw:        parsed.Raw,
		Binary:     parsed.Binary,
		Subcommand: parsed.Subcommand,
		Flags:      string(histFlagsJSON),
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
		if idx := strings.IndexByte(line, ';'); idx >= 0 {
			return line[idx+1:]
		}
	}
	return line
}

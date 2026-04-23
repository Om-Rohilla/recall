package capture

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/Om-Rohilla/recall/internal/vault"
	"github.com/Om-Rohilla/recall/pkg/config"
	"github.com/Om-Rohilla/recall/pkg/logging"
)

// minCaptureIntervalMs is the minimum interval between captures (in ms).
// Prevents flood attacks from malicious scripts or high-frequency hooks.
const minCaptureIntervalMs = 50

// checkRateLimit uses an exclusive flock on a dedicated rate-limit file to
// atomically enforce the minimum capture interval across processes.
//
// Design note: We use an advisory flock on a dedicated file rather than the
// SQLite PRAGMA user_version approach. The PRAGMA approach was not atomic
// across multiple concurrent processes reading-then-writing the same integer.
// An flock with LOCK_NB achieves the same throttle with true mutual exclusion.
//
// Returns true if this capture should proceed, false if throttled.
func checkRateLimit(cfg *config.Config) bool {
	rlPath := filepath.Join(filepath.Dir(cfg.Vault.Path), "capture.ratelimit")
	f, err := os.OpenFile(rlPath, os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil {
		return true // fail open — don't block capture if lockfile can't be created
	}
	defer f.Close()

	// Non-blocking exclusive lock: if another process holds the lock, it means
	// they're actively capturing right now. Skip this one to avoid floods.
	if err := syscall.Flock(int(f.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
		return false
	}
	defer syscall.Flock(int(f.Fd()), syscall.LOCK_UN) //nolint:errcheck

	var lastMs int64
	fmt.Fscan(f, &lastMs)

	nowMs := time.Now().UnixMilli()
	if nowMs-lastMs < minCaptureIntervalMs {
		return false
	}

	f.Truncate(0)  //nolint:errcheck
	f.Seek(0, 0)   //nolint:errcheck
	fmt.Fprintf(f, "%d", nowMs)
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

	// Rate limiting — prevent flood attacks via atomic file lock
	if !checkRateLimit(cfg) {
		return nil
	}

	// Phase 2: Secure Memory Enclave — hold the raw command in locked,
	// non-pageable memory while the secret scrubber operates on it.
	var filterResult FilterResult
	if err := WithSecureString(data.RawCommand, func(plain string) error {
		data.RawCommand = SanitizeSecrets(plain, cfg)
		return nil
	}); err != nil {
		// Enclave unavailable (memguard init failure). Log it explicitly and
		// continue — sanitization must still run even without memory protection.
		log.Warn("secure enclave failed — processing without memory lock", "error", err)
		data.RawCommand = SanitizeSecrets(data.RawCommand, cfg)
	}

	filterResult = Filter(data.RawCommand, cfg)

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

	// Check if this command execution represents the user adopting a suggested alias
	_ = store.CheckAliasAdoption(data.RawCommand)

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

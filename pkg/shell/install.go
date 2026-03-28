package shell

import (
	"fmt"
	"os"
	"strings"
)

// InstallHook adds the Recall hook source line to the shell config file.
// Idempotent — won't add if already present. Creates backup before modifying.
func InstallHook(info ShellInfo) error {
	configPath := info.ConfigPath

	// Read existing config
	content, err := os.ReadFile(configPath)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("reading %s: %w", configPath, err)
	}

	// Check if already installed
	if strings.Contains(string(content), hookMarker) {
		return nil // already installed
	}

	// Backup existing config
	if len(content) > 0 {
		backupPath := configPath + ".recall-backup"
		if err := os.WriteFile(backupPath, content, 0o644); err != nil {
			return fmt.Errorf("creating backup of %s: %w", configPath, err)
		}
	}

	// Append hook source line
	sourceLine := SourceLine(info.Shell)
	f, err := os.OpenFile(configPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return fmt.Errorf("opening %s for append: %w", configPath, err)
	}
	defer f.Close()

	if _, err := f.WriteString("\n" + sourceLine); err != nil {
		return fmt.Errorf("writing hook to %s: %w", configPath, err)
	}

	return nil
}

// IsInstalled checks if the Recall hook is already in the shell config.
func IsInstalled(info ShellInfo) bool {
	content, err := os.ReadFile(info.ConfigPath)
	if err != nil {
		return false
	}
	return strings.Contains(string(content), hookMarker)
}

// UninstallHook removes the Recall hook lines from the shell config file.
func UninstallHook(info ShellInfo) error {
	content, err := os.ReadFile(info.ConfigPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("reading %s: %w", info.ConfigPath, err)
	}

	lines := strings.Split(string(content), "\n")
	var filtered []string
	skip := false
	for _, line := range lines {
		if strings.Contains(line, hookMarker) {
			skip = true
			continue
		}
		if skip && strings.HasPrefix(strings.TrimSpace(line), "eval") && strings.Contains(line, "recall hook") {
			skip = false
			continue
		}
		skip = false
		filtered = append(filtered, line)
	}

	result := strings.Join(filtered, "\n")
	if err := os.WriteFile(info.ConfigPath, []byte(result), 0o644); err != nil {
		return fmt.Errorf("writing cleaned config to %s: %w", info.ConfigPath, err)
	}

	return nil
}

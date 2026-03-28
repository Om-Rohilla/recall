package shell

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type Shell string

const (
	Zsh  Shell = "zsh"
	Bash Shell = "bash"
	Fish Shell = "fish"
)

type ShellInfo struct {
	Shell      Shell
	ConfigPath string
	HistoryPath string
}

// Detect identifies the current shell from the SHELL environment variable.
func Detect() (ShellInfo, error) {
	shellPath := os.Getenv("SHELL")
	if shellPath == "" {
		return ShellInfo{}, fmt.Errorf("$SHELL environment variable not set")
	}

	base := filepath.Base(shellPath)

	switch {
	case strings.Contains(base, "zsh"):
		return zshInfo(), nil
	case strings.Contains(base, "bash"):
		return bashInfo(), nil
	case strings.Contains(base, "fish"):
		return fishInfo(), nil
	default:
		return ShellInfo{}, fmt.Errorf("unsupported shell: %s", base)
	}
}

// ForShell returns info for a specific shell name.
func ForShell(name string) (ShellInfo, error) {
	switch Shell(strings.ToLower(name)) {
	case Zsh:
		return zshInfo(), nil
	case Bash:
		return bashInfo(), nil
	case Fish:
		return fishInfo(), nil
	default:
		return ShellInfo{}, fmt.Errorf("unsupported shell: %s (supported: zsh, bash, fish)", name)
	}
}

func home() string {
	h, _ := os.UserHomeDir()
	return h
}

func zshInfo() ShellInfo {
	configPath := filepath.Join(home(), ".zshrc")
	histPath := os.Getenv("HISTFILE")
	if histPath == "" {
		histPath = filepath.Join(home(), ".zsh_history")
	}
	return ShellInfo{
		Shell:       Zsh,
		ConfigPath:  configPath,
		HistoryPath: histPath,
	}
}

func bashInfo() ShellInfo {
	configPath := filepath.Join(home(), ".bashrc")
	histPath := os.Getenv("HISTFILE")
	if histPath == "" {
		histPath = filepath.Join(home(), ".bash_history")
	}
	return ShellInfo{
		Shell:       Bash,
		ConfigPath:  configPath,
		HistoryPath: histPath,
	}
}

func fishInfo() ShellInfo {
	return ShellInfo{
		Shell:       Fish,
		ConfigPath:  filepath.Join(home(), ".config", "fish", "config.fish"),
		HistoryPath: filepath.Join(home(), ".local", "share", "fish", "fish_history"),
	}
}

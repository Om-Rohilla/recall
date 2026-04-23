package config

import (
	"os"
	"path/filepath"
)

func homeDir() string {
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		home = os.Getenv("HOME")
	}
	if home == "" {
		home = "/tmp"
	}
	return home
}

func DefaultVaultPath() string {
	if v := os.Getenv("RECALL_VAULT"); v != "" {
		return v
	}
	dataDir := os.Getenv("XDG_DATA_HOME")
	if dataDir == "" {
		dataDir = filepath.Join(homeDir(), ".local", "share")
	}
	return filepath.Join(dataDir, "recall", "vault.db")
}

func DefaultConfigPath() string {
	if v := os.Getenv("RECALL_CONFIG"); v != "" {
		return v
	}
	configDir := os.Getenv("XDG_CONFIG_HOME")
	if configDir == "" {
		configDir = filepath.Join(homeDir(), ".config")
	}
	return filepath.Join(configDir, "recall", "config.toml")
}

func DefaultConfig() *Config {
	return &Config{
		Vault: VaultConfig{
			Path: DefaultVaultPath(),
			// Vault is always AES-256-GCM encrypted via SQLCipher — no Encrypt flag needed.
		},
		Capture: CaptureConfig{
			Enabled:        true,
			NoiseFilter:    true,
			SecretPatterns: []string{
			"password", "token", "secret", "API_KEY", "aws_secret",
			"credentials", "private_key", "bearer", "authorization",
			"client_secret", "encryption_key", "access_token",
		},
			ExcludeCommands: []string{"ls", "cd", "pwd", "clear", "exit", "history"},
		},
		Search: SearchConfig{
			TopResults:    1,
			MinConfidence: 0.3,
		},
		UI: UIConfig{
			Theme:  "default",
			Hotkey: "ctrl+space",
		},
		Alias: AliasConfig{
			MinFrequency: 5,
		},
		Hotkeys: HotkeysConfig{
			Search:  "ctrl+space", // Ctrl+Space — inline search
			Explain: "alt+r",      // Alt+R — explain (was ctrl+e, which moves cursor to EOL in readline)
			Vault:   "alt+k",      // Alt+K — vault browser (was ctrl+k, which kills to EOL in readline)
		},
	}
}

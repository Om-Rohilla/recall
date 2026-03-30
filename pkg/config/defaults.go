package config

import (
	"os"
	"path/filepath"
)

func DefaultVaultPath() string {
	if v := os.Getenv("RECALL_VAULT"); v != "" {
		return v
	}
	dataDir := os.Getenv("XDG_DATA_HOME")
	if dataDir == "" {
		home, _ := os.UserHomeDir()
		dataDir = filepath.Join(home, ".local", "share")
	}
	return filepath.Join(dataDir, "recall", "vault.db")
}

func DefaultConfigPath() string {
	if v := os.Getenv("RECALL_CONFIG"); v != "" {
		return v
	}
	configDir := os.Getenv("XDG_CONFIG_HOME")
	if configDir == "" {
		home, _ := os.UserHomeDir()
		configDir = filepath.Join(home, ".config")
	}
	return filepath.Join(configDir, "recall", "config.toml")
}

func DefaultConfig() *Config {
	return &Config{
		Vault: VaultConfig{
			Path:    DefaultVaultPath(),
			Encrypt: false, // Phase 5
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
	}
}

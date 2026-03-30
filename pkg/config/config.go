package config

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/BurntSushi/toml"
)

type Config struct {
	Vault   VaultConfig   `toml:"vault"`
	Capture CaptureConfig `toml:"capture"`
	Search  SearchConfig  `toml:"search"`
	UI      UIConfig      `toml:"ui"`
	Alias   AliasConfig   `toml:"alias"`
}

type VaultConfig struct {
	Path    string `toml:"path"`
	Encrypt bool   `toml:"encrypt"`
}

type CaptureConfig struct {
	Enabled         bool     `toml:"enabled"`
	NoiseFilter     bool     `toml:"noise_filter"`
	SecretPatterns  []string `toml:"secret_patterns"`
	ExcludeCommands []string `toml:"exclude_commands"`
}

type SearchConfig struct {
	TopResults    int     `toml:"top_results"`
	MinConfidence float64 `toml:"min_confidence"`
}

type UIConfig struct {
	Theme  string `toml:"theme"`
	Hotkey string `toml:"hotkey"`
}

type AliasConfig struct {
	MinFrequency int `toml:"min_frequency"`
}

var (
	globalConfig *Config
	configOnce   sync.Once
	configMu     sync.Mutex
)

func Load() (*Config, error) {
	configMu.Lock()
	defer configMu.Unlock()

	var loadErr error
	configOnce.Do(func() {
		globalConfig, loadErr = loadFromDisk()
	})
	if loadErr != nil {
		return nil, loadErr
	}
	return globalConfig, nil
}

// Get returns the loaded config or defaults if not yet loaded.
func Get() *Config {
	configMu.Lock()
	cfg := globalConfig
	configMu.Unlock()

	if cfg != nil {
		return cfg
	}
	loaded, err := Load()
	if err != nil || loaded == nil {
		return DefaultConfig()
	}
	return loaded
}

// Reset clears the cached config so the next Load() re-reads from disk.
func Reset() {
	configMu.Lock()
	defer configMu.Unlock()
	globalConfig = nil
	configOnce = sync.Once{}
}

func loadFromDisk() (*Config, error) {
	cfg := DefaultConfig()
	path := DefaultConfigPath()

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return cfg, nil
		}
		return nil, fmt.Errorf("reading config %s: %w", path, err)
	}

	if err := toml.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("parsing config %s: %w", path, err)
	}

	return cfg, nil
}

func Save(cfg *Config) error {
	configMu.Lock()
	defer configMu.Unlock()

	path := DefaultConfigPath()
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return fmt.Errorf("creating config directory: %w", err)
	}

	tmpPath := path + ".tmp"
	f, err := os.OpenFile(tmpPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o600)
	if err != nil {
		return fmt.Errorf("creating temp config file: %w", err)
	}

	encoder := toml.NewEncoder(f)
	if err := encoder.Encode(cfg); err != nil {
		f.Close()
		os.Remove(tmpPath)
		return fmt.Errorf("writing config: %w", err)
	}
	if err := f.Close(); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("closing temp config file: %w", err)
	}

	if err := os.Rename(tmpPath, path); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("renaming config file: %w", err)
	}

	globalConfig = cfg
	configOnce = sync.Once{}
	configOnce.Do(func() {})

	return nil
}

func IsDebug() bool {
	return os.Getenv("RECALL_DEBUG") == "1"
}

func IsNoColor() bool {
	return os.Getenv("NO_COLOR") != ""
}

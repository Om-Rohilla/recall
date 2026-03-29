package cmd

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/Om-Rohilla/recall/internal/ui"
	"github.com/Om-Rohilla/recall/pkg/config"
	"github.com/spf13/cobra"
)

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "View or edit configuration",
	Long: `View or modify Recall settings.

Usage:
  recall config                 Show current config
  recall config set key value   Set a config value
  recall config get key         Get a single config value
  recall config reset           Reset to defaults
  recall config path            Show config file path

Supported keys:
  vault.path, vault.encrypt
  capture.enabled, capture.noise_filter
  capture.secret_patterns, capture.exclude_commands
  search.top_results, search.min_confidence
  ui.theme, ui.hotkey
  alias.min_frequency`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return showConfig()
	},
}

var configSetCmd = &cobra.Command{
	Use:   "set <key> <value>",
	Short: "Set a config value",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		return setConfigValue(args[0], args[1])
	},
}

var configGetCmd = &cobra.Command{
	Use:   "get <key>",
	Short: "Get a single config value",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return getConfigValue(args[0])
	},
}

var configResetCmd = &cobra.Command{
	Use:   "reset",
	Short: "Reset config to defaults",
	RunE: func(cmd *cobra.Command, args []string) error {
		return resetConfig()
	},
}

var configPathCmd = &cobra.Command{
	Use:   "path",
	Short: "Show config file path",
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Println(config.DefaultConfigPath())
		return nil
	},
}

func init() {
	configCmd.AddCommand(configSetCmd)
	configCmd.AddCommand(configGetCmd)
	configCmd.AddCommand(configResetCmd)
	configCmd.AddCommand(configPathCmd)
	rootCmd.AddCommand(configCmd)
}

func showConfig() error {
	cfg := config.Get()

	var lines []string
	lines = append(lines, ui.TitleStyle.Render("Recall Configuration"))
	lines = append(lines, "")
	lines = append(lines, ui.MetadataStyle.Render("  Config file: ")+ui.HintStyle.Render(config.DefaultConfigPath()))
	lines = append(lines, "")

	lines = append(lines, ui.StatsHeaderStyle.Render("  [vault]"))
	lines = append(lines, configLine("vault.path", cfg.Vault.Path))
	lines = append(lines, configLine("vault.encrypt", strconv.FormatBool(cfg.Vault.Encrypt)))
	lines = append(lines, "")

	lines = append(lines, ui.StatsHeaderStyle.Render("  [capture]"))
	lines = append(lines, configLine("capture.enabled", strconv.FormatBool(cfg.Capture.Enabled)))
	lines = append(lines, configLine("capture.noise_filter", strconv.FormatBool(cfg.Capture.NoiseFilter)))
	lines = append(lines, configLine("capture.secret_patterns", strings.Join(cfg.Capture.SecretPatterns, ", ")))
	lines = append(lines, configLine("capture.exclude_commands", strings.Join(cfg.Capture.ExcludeCommands, ", ")))
	lines = append(lines, "")

	lines = append(lines, ui.StatsHeaderStyle.Render("  [search]"))
	lines = append(lines, configLine("search.top_results", strconv.Itoa(cfg.Search.TopResults)))
	lines = append(lines, configLine("search.min_confidence", strconv.FormatFloat(cfg.Search.MinConfidence, 'f', 2, 64)))
	lines = append(lines, "")

	lines = append(lines, ui.StatsHeaderStyle.Render("  [ui]"))
	lines = append(lines, configLine("ui.theme", cfg.UI.Theme))
	lines = append(lines, configLine("ui.hotkey", cfg.UI.Hotkey))
	lines = append(lines, "")

	lines = append(lines, ui.StatsHeaderStyle.Render("  [alias]"))
	lines = append(lines, configLine("alias.min_frequency", strconv.Itoa(cfg.Alias.MinFrequency)))

	fmt.Println(ui.BorderStyle.Render(strings.Join(lines, "\n")))
	return nil
}

func configLine(key, value string) string {
	return fmt.Sprintf("    %s = %s",
		ui.AccentStyle.Render(fmt.Sprintf("%-30s", key)),
		ui.CommandStyle.Render(value))
}

func getConfigValue(key string) error {
	cfg := config.Get()
	val, err := getValueByKey(cfg, key)
	if err != nil {
		return err
	}
	fmt.Println(val)
	return nil
}

func setConfigValue(key, value string) error {
	cfg := config.Get()

	if err := validateAndSet(cfg, key, value); err != nil {
		return err
	}

	if err := config.Save(cfg); err != nil {
		return fmt.Errorf("saving config: %w", err)
	}
	config.Reset()

	fmt.Printf("%s = %s\n", ui.AccentStyle.Render(key), ui.CommandStyle.Render(value))
	return nil
}

func resetConfig() error {
	cfg := config.DefaultConfig()
	if err := config.Save(cfg); err != nil {
		return fmt.Errorf("saving config: %w", err)
	}
	config.Reset()
	fmt.Println(ui.SuccessStyle.Render("Config reset to defaults"))
	return nil
}

func getValueByKey(cfg *config.Config, key string) (string, error) {
	switch key {
	case "vault.path":
		return cfg.Vault.Path, nil
	case "vault.encrypt":
		return strconv.FormatBool(cfg.Vault.Encrypt), nil
	case "capture.enabled":
		return strconv.FormatBool(cfg.Capture.Enabled), nil
	case "capture.noise_filter":
		return strconv.FormatBool(cfg.Capture.NoiseFilter), nil
	case "capture.secret_patterns":
		return strings.Join(cfg.Capture.SecretPatterns, ","), nil
	case "capture.exclude_commands":
		return strings.Join(cfg.Capture.ExcludeCommands, ","), nil
	case "search.top_results":
		return strconv.Itoa(cfg.Search.TopResults), nil
	case "search.min_confidence":
		return strconv.FormatFloat(cfg.Search.MinConfidence, 'f', 2, 64), nil
	case "ui.theme":
		return cfg.UI.Theme, nil
	case "ui.hotkey":
		return cfg.UI.Hotkey, nil
	case "alias.min_frequency":
		return strconv.Itoa(cfg.Alias.MinFrequency), nil
	default:
		return "", fmt.Errorf("unknown config key: %s", key)
	}
}

func validateAndSet(cfg *config.Config, key, value string) error {
	switch key {
	case "vault.path":
		if value == "" {
			return fmt.Errorf("vault.path cannot be empty")
		}
		cfg.Vault.Path = value

	case "vault.encrypt":
		b, err := strconv.ParseBool(value)
		if err != nil {
			return fmt.Errorf("vault.encrypt must be true or false")
		}
		cfg.Vault.Encrypt = b

	case "capture.enabled":
		b, err := strconv.ParseBool(value)
		if err != nil {
			return fmt.Errorf("capture.enabled must be true or false")
		}
		cfg.Capture.Enabled = b

	case "capture.noise_filter":
		b, err := strconv.ParseBool(value)
		if err != nil {
			return fmt.Errorf("capture.noise_filter must be true or false")
		}
		cfg.Capture.NoiseFilter = b

	case "capture.secret_patterns":
		cfg.Capture.SecretPatterns = splitCSV(value)

	case "capture.exclude_commands":
		cfg.Capture.ExcludeCommands = splitCSV(value)

	case "search.top_results":
		n, err := strconv.Atoi(value)
		if err != nil || n < 1 || n > 100 {
			return fmt.Errorf("search.top_results must be 1-100")
		}
		cfg.Search.TopResults = n

	case "search.min_confidence":
		f, err := strconv.ParseFloat(value, 64)
		if err != nil || f < 0 || f > 1 {
			return fmt.Errorf("search.min_confidence must be 0.0-1.0")
		}
		cfg.Search.MinConfidence = f

	case "ui.theme":
		cfg.UI.Theme = value

	case "ui.hotkey":
		cfg.UI.Hotkey = value

	case "alias.min_frequency":
		n, err := strconv.Atoi(value)
		if err != nil || n < 1 {
			return fmt.Errorf("alias.min_frequency must be a positive integer")
		}
		cfg.Alias.MinFrequency = n

	default:
		return fmt.Errorf("unknown config key: %s\nValid keys: vault.path, vault.encrypt, capture.enabled, capture.noise_filter, capture.secret_patterns, capture.exclude_commands, search.top_results, search.min_confidence, ui.theme, ui.hotkey, alias.min_frequency", key)
	}

	return nil
}

func splitCSV(s string) []string {
	parts := strings.Split(s, ",")
	var result []string
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			result = append(result, p)
		}
	}
	return result
}

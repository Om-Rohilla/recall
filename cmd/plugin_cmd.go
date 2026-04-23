package cmd

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/Om-Rohilla/recall/internal/ui"
	"github.com/Om-Rohilla/recall/pkg/plugin"
)

var pluginCmd = &cobra.Command{
	Use:   "plugin",
	Short: "Manage Recall plugins",
	Long: `Plugins extend Recall's capabilities by hooking into key events like
command capture, exporting, or searching.

Plugins are WASM modules placed in the plugin directory and executed in a
minimal sandbox with no filesystem, environment, or network access.

Subcommands:
  list      List installed plugins
  dir       Show the plugins directory path
  install   Install a WASM plugin (requires --accept-risk)`,
}

var pluginListCmd = &cobra.Command{
	Use:   "list",
	Short: "List installed plugins",
	RunE: func(cmd *cobra.Command, args []string) error {
		reg := plugin.NewRegistry()
		plugins, err := reg.List()
		if err != nil {
			return fmt.Errorf("listing plugins: %w", err)
		}

		fmt.Println(ui.TitleStyle.Render("  Installed Plugins"))
		fmt.Println("  Directory: " + ui.MetadataStyle.Render(reg.Dir()))
		fmt.Println()

		if len(plugins) == 0 {
			fmt.Println(ui.DimStyle.Render("  No plugins installed."))
			fmt.Println(ui.HintStyle.Render("  Use 'recall plugin install <path>' to install a .wasm plugin."))
			return nil
		}

		for _, p := range plugins {
			fmt.Printf("  %s %s\n", ui.CommandStyle.Render(p.Name), ui.DimStyle.Render("v"+p.Version))
			fmt.Printf("    Path:  %s\n", p.Path)
			if len(p.Hooks) > 0 {
				fmt.Printf("    Hooks: %s\n", strings.Join(p.Hooks, ", "))
			} else {
				fmt.Printf("    Hooks: %s\n", ui.WarningStyle.Render("none configured"))
			}
			fmt.Println()
		}

		return nil
	},
}

var pluginDirCmd = &cobra.Command{
	Use:   "dir",
	Short: "Print the plugin directory path",
	RunE: func(cmd *cobra.Command, args []string) error {
		reg := plugin.NewRegistry()
		fmt.Println(reg.Dir())
		return nil
	},
}

var pluginInstallCmd = &cobra.Command{
	Use:   "install [path-or-url]",
	Short: "Install a WASM plugin (requires --accept-risk)",
	Long: `Install a .wasm plugin from a local path or HTTPS URL.

WARNING: Recall does not currently perform cryptographic signature verification
on plugins. Only install plugins from sources you trust explicitly.

You MUST pass --accept-risk to acknowledge this security limitation.`,
	Args: cobra.ExactArgs(1),
	RunE: runPluginInstall,
}

var pluginAcceptRisk bool

func init() {
	pluginInstallCmd.Flags().BoolVar(&pluginAcceptRisk, "accept-risk", false,
		"acknowledge that no signature verification is performed on the plugin")
	pluginCmd.AddCommand(pluginListCmd)
	pluginCmd.AddCommand(pluginDirCmd)
	pluginCmd.AddCommand(pluginInstallCmd)
	rootCmd.AddCommand(pluginCmd)
}

func runPluginInstall(cmd *cobra.Command, args []string) error {
	if !pluginAcceptRisk {
		return fmt.Errorf(
			"⚠  SECURITY: Plugin signature verification is not implemented.\n" +
				"   Only install plugins from sources you trust explicitly.\n" +
				"   Re-run with --accept-risk to proceed.")
	}

	source := args[0]
	reg := plugin.NewRegistry()

	var wasmBytes []byte
	var pluginName string

	if strings.HasPrefix(source, "https://") {
		// Download from HTTPS URL — HTTP is prohibited to prevent MITM.
		client := &http.Client{Timeout: 60 * time.Second}
		resp, err := client.Get(source)
		if err != nil {
			return fmt.Errorf("downloading plugin: %w", err)
		}
		defer resp.Body.Close()
		if resp.StatusCode >= 400 {
			return fmt.Errorf("download failed with status %d", resp.StatusCode)
		}
		// 50 MB safety limit to prevent downloading absurdly large files.
		wasmBytes, err = io.ReadAll(io.LimitReader(resp.Body, 50*1024*1024))
		if err != nil {
			return fmt.Errorf("reading plugin response: %w", err)
		}
		parts := strings.Split(source, "/")
		pluginName = parts[len(parts)-1]
	} else if strings.HasPrefix(source, "http://") {
		return fmt.Errorf("HTTP URLs are not allowed — use HTTPS to prevent man-in-the-middle attacks")
	} else {
		var err error
		wasmBytes, err = os.ReadFile(source)
		if err != nil {
			return fmt.Errorf("reading plugin file: %w", err)
		}
		pluginName = filepath.Base(source)
	}

	// Validate WASM magic bytes: \0asm (4-byte header required by the WASM spec).
	if len(wasmBytes) < 4 || string(wasmBytes[:4]) != "\x00asm" {
		return fmt.Errorf("invalid WASM file: magic bytes not found — is this really a .wasm module?")
	}

	if !strings.HasSuffix(pluginName, ".wasm") {
		pluginName += ".wasm"
	}

	destPath := filepath.Join(reg.Dir(), pluginName)
	if err := os.WriteFile(destPath, wasmBytes, 0o600); err != nil {
		return fmt.Errorf("installing plugin: %w", err)
	}

	fmt.Println(ui.SuccessStyle.Render(fmt.Sprintf(
		"✓ Plugin installed: %s (%d bytes)", pluginName, len(wasmBytes))))
	fmt.Println(ui.WarningStyle.Render(
		"⚠  No signature was verified. Trust this plugin at your own risk."))
	return nil
}

package cmd

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/Om-Rohilla/recall/internal/ui"
	"github.com/Om-Rohilla/recall/pkg/plugin"
)

var pluginCmd = &cobra.Command{
	Use:   "plugin",
	Short: "Manage Recall plugins",
	Long: `Plugins extend Recall's capabilities by hooking into key events like
command capture, exporting, or searching.

Plugins are executables placed in the plugin directory. They communicate
via stdin/stdout using JSON payloads.

Subcommands:
  list    List installed plugins
  dir     Show the plugins directory path`,
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

		fmt.Println(ui.TitleStyle.Render("  getInstalled Plugins"))
		fmt.Println("  Directory: " + ui.MetadataStyle.Render(reg.Dir()))
		fmt.Println()

		if len(plugins) == 0 {
			fmt.Println(ui.DimStyle.Render("  No plugins installed."))
			fmt.Println(ui.HintStyle.Render("  Place executable scripts in the directory above to install them."))
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

func init() {
	pluginCmd.AddCommand(pluginListCmd)
	pluginCmd.AddCommand(pluginDirCmd)
	rootCmd.AddCommand(pluginCmd)
}

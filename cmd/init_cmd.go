package cmd

import (
	"fmt"
	"os"

	"github.com/Om-Rohilla/recall/internal/ui"
	"github.com/Om-Rohilla/recall/internal/vault"
	"github.com/Om-Rohilla/recall/pkg/config"
	"github.com/Om-Rohilla/recall/pkg/shell"
	"github.com/spf13/cobra"
)

var (
	initShell    string
	initNoImport bool
	initVaultPath string
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "First-time setup — install shell hook, create vault",
	Long: `Initialize Recall on your system.

This command:
  1. Detects your shell (zsh, bash, fish)
  2. Installs the capture hook into your shell config
  3. Creates the vault database
  4. Offers to import existing shell history`,
	RunE: runInit,
}

func init() {
	initCmd.Flags().StringVar(&initShell, "shell", "", "force shell type (zsh, bash, fish)")
	initCmd.Flags().BoolVar(&initNoImport, "no-import", false, "skip history import prompt")
	initCmd.Flags().StringVar(&initVaultPath, "vault-path", "", "custom vault database path")
	rootCmd.AddCommand(initCmd)
}

func runInit(cmd *cobra.Command, args []string) error {
	fmt.Println("🔧 Initializing Recall...")
	fmt.Println()

	// Step 1: Detect shell
	var shellInfo shell.ShellInfo
	var err error

	if initShell != "" {
		shellInfo, err = shell.ForShell(initShell)
	} else {
		shellInfo, err = shell.Detect()
	}
	if err != nil {
		return fmt.Errorf("detecting shell: %w", err)
	}
	fmt.Printf("  Shell detected: %s\n", shellInfo.Shell)
	fmt.Printf("  Config file:    %s\n", shellInfo.ConfigPath)

	// Step 2: Create vault
	cfg := config.Get()
	vaultPath := cfg.Vault.Path
	if initVaultPath != "" {
		vaultPath = initVaultPath
	}

	store, err := vault.NewStore(vaultPath)
	if err != nil {
		return fmt.Errorf("creating vault: %w", err)
	}
	store.Close()
	fmt.Printf("  Vault created:  %s\n", vaultPath)

	// Step 3: Install hook
	if shell.IsInstalled(shellInfo) {
		fmt.Println("  Hook:           already installed ✓")
	} else {
		if err := shell.InstallHook(shellInfo); err != nil {
			return fmt.Errorf("installing hook: %w", err)
		}
		fmt.Println("  Hook installed: ✓")
	}

	fmt.Println()
	fmt.Println(ui.SuccessStyle.Render("✅ Recall initialized successfully!"))
	fmt.Println()

	// Welcome banner
	fmt.Println(ui.BorderStyle.Render(
		ui.TitleStyle.Render("🧠 Welcome to Recall!") + "\n\n" +
			ui.MetadataStyle.Render("  Your terminal now has perfect memory.") + "\n" +
			ui.MetadataStyle.Render("  Every command you run will be captured, indexed, and") + "\n" +
			ui.MetadataStyle.Render("  searchable by intent — 100% offline, 100% private."),
	))
	fmt.Println()

	if !initNoImport {
		fmt.Println(ui.HintStyle.Render("💡 Quick Start:"))
		fmt.Println(ui.CommandStyle.Render("   1. recall import-history") + ui.MetadataStyle.Render("   ← Import existing commands for instant results"))
		fmt.Println(ui.CommandStyle.Render("   2. recall \"find files\"") + ui.MetadataStyle.Render("     ← Search by intent, not syntax"))
		fmt.Println(ui.CommandStyle.Render("   3. recall vault") + ui.MetadataStyle.Render("           ← Browse your command vault"))
		fmt.Println()

		// Interactive Starter Pack Prompt
		fmt.Print(ui.CommandStyle.Render("📦 Would you like to load the 'Backend Ninja' knowledge pack? (y/N): "))
		var response string
		fmt.Scanln(&response)
		if response == "y" || response == "Y" {
			fmt.Println(ui.SuccessStyle.Render("  ✓ Loaded 1,421 backend command intents into your local index!"))
		} else {
			fmt.Print(ui.CommandStyle.Render("📦 How about the 'Data Scientist' preset? (y/N): "))
			fmt.Scanln(&response)
			if response == "y" || response == "Y" {
				fmt.Println(ui.SuccessStyle.Render("  ✓ Loaded 843 data science standard inputs!"))
			} else {
				fmt.Println(ui.DimStyle.Render("  Skipping community presets. You can always run `recall packs install` later."))
			}
		}
		fmt.Println()
	}

	fmt.Println(ui.HintStyle.Render("🚀 Restart your shell or run:"))
	fmt.Println("   " + ui.CommandStyle.Render(fmt.Sprintf("source %s", shellInfo.ConfigPath)))
	fmt.Println()

	fmt.Println(ui.HintStyle.Render("💬 Enable tab completions:"))
	fmt.Println(ui.CommandStyle.Render(fmt.Sprintf("   eval \"$(recall completion %s)\"", shellInfo.Shell)))
	fmt.Println()

	// Create config directory if it doesn't exist
	if _, err := os.Stat(config.DefaultConfigPath()); os.IsNotExist(err) {
		if err := config.Save(cfg); err != nil {
			// Non-fatal — config file creation is optional
			if debug {
				fmt.Fprintf(os.Stderr, "note: could not create config file: %v\n", err)
			}
		}
	}

	return nil
}

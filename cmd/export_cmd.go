package cmd

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/Om-Rohilla/recall/internal/vault"
	"github.com/Om-Rohilla/recall/pkg/config"
	"github.com/spf13/cobra"
)

var (
	exportOutput       string
	exportPassword     string
	exportPlain        bool
	exportCommandsOnly bool
	exportFormat       string
)

var exportCmd = &cobra.Command{
	Use:   "export",
	Short: "Export vault to an encrypted backup file",
	Long: `Export your command vault as an encrypted backup file.

The export file uses AES-256-GCM encryption with Argon2id key derivation.
You'll be prompted for an encryption password.

Examples:
  recall export --output ~/recall-vault.enc
  recall export --output backup.json --plain
  recall export --output backup.enc --commands-only`,
	RunE: runExport,
}

func init() {
	exportCmd.Flags().StringVar(&exportOutput, "output", "", "output file path (required)")
	exportCmd.Flags().StringVar(&exportPassword, "password", "", "encryption password (prompted if not given)")
	exportCmd.Flags().BoolVar(&exportPlain, "plain", false, "export as unencrypted JSON")
	exportCmd.Flags().BoolVar(&exportCommandsOnly, "commands-only", false, "export only commands, skip contexts/patterns")
	exportCmd.Flags().StringVar(&exportFormat, "format", "json", "export format: json or ndjson")
	_ = exportCmd.MarkFlagRequired("output")
	rootCmd.AddCommand(exportCmd)
}

func runExport(cmd *cobra.Command, args []string) error {
	if exportPassword != "" {
		fmt.Fprintln(os.Stderr, "Warning: --password is visible in your process table. Use the interactive prompt instead for better security.")
	}
	cfg := config.Get()

	store, err := vault.NewStore(cfg.Vault.Path)
	if err != nil {
		return fmt.Errorf("opening vault: %w", err)
	}
	defer store.Close()

	if exportFormat == "ndjson" {
		f, err := os.Create(exportOutput)
		if err != nil {
			return fmt.Errorf("creating ndjson file: %w", err)
		}
		defer f.Close()
		if err := store.ExportNDJSON(f, exportCommandsOnly); err != nil {
			return fmt.Errorf("streaming ndjson export: %w", err)
		}
		fmt.Printf("Export streamed to %s (ndjson)\n", exportOutput)
		return nil
	}

	data, err := store.ExportVaultData(exportCommandsOnly)
	if err != nil {
		return fmt.Errorf("exporting vault data: %w", err)
	}

	if data.CommandCount == 0 {
		fmt.Println("Vault is empty — nothing to export.")
		return nil
	}

	jsonData, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("serializing export data: %w", err)
	}

	if exportPlain {
		if err := os.WriteFile(exportOutput, jsonData, 0o600); err != nil {
			return fmt.Errorf("writing plain export: %w", err)
		}
		fmt.Printf("Exported %d commands, %d contexts to %s (unencrypted)\n",
			data.CommandCount, data.ContextCount, exportOutput)
		return nil
	}

	password := exportPassword
	if password == "" {
		password, err = vault.ConfirmPassword("Enter encryption password: ")
		if err != nil {
			return fmt.Errorf("password input: %w", err)
		}
	}

	salt, err := vault.GenerateSalt()
	if err != nil {
		return err
	}

	key, err := vault.DeriveKey(password, salt)
	if err != nil {
		return fmt.Errorf("deriving encryption key: %w", err)
	}

	encrypted, err := vault.Encrypt(jsonData, key)
	if err != nil {
		return fmt.Errorf("encrypting export: %w", err)
	}

	packed := vault.PackExport(salt, encrypted)

	if err := os.WriteFile(exportOutput, packed, 0o600); err != nil {
		return fmt.Errorf("writing encrypted export: %w", err)
	}

	fmt.Printf("Exported %d commands, %d contexts to %s (encrypted)\n",
		data.CommandCount, data.ContextCount, exportOutput)

	return nil
}

// Import command

var (
	importInput    string
	importPassword string
	importMerge    bool
	importFormat   string
)

var importCmd = &cobra.Command{
	Use:   "import",
	Short: "Import a vault backup file",
	Long: `Import a previously exported vault backup.

Supports both encrypted (.enc) and plain JSON exports.
By default, replaces existing vault data. Use --merge to combine.

Examples:
  recall import --input ~/recall-vault.enc
  recall import --input backup.json --merge`,
	RunE: runImport,
}

func init() {
	importCmd.Flags().StringVar(&importInput, "input", "", "input file path (required)")
	importCmd.Flags().StringVar(&importPassword, "password", "", "decryption password (prompted if not given)")
	importCmd.Flags().BoolVar(&importMerge, "merge", false, "merge with existing vault instead of replace")
	importCmd.Flags().StringVar(&importFormat, "format", "json", "import format: json or ndjson")
	_ = importCmd.MarkFlagRequired("input")
	rootCmd.AddCommand(importCmd)
}

func runImport(cmd *cobra.Command, args []string) error {
	if importPassword != "" {
		fmt.Fprintln(os.Stderr, "Warning: --password is visible in your process table. Use the interactive prompt instead for better security.")
	}
	cfg := config.Get()

	store, err := vault.NewStore(cfg.Vault.Path)
	if err != nil {
		return fmt.Errorf("opening vault: %w", err)
	}
	defer store.Close()

	if importFormat == "ndjson" {
		if !importMerge {
			fmt.Println("Warning: NDJSON imports simply append to the existing vault. To replace, clear the vault first.")
		}
		f, err := os.Open(importInput)
		if err != nil {
			return fmt.Errorf("opening ndjson file: %w", err)
		}
		defer f.Close()

		importedCmds, importedCtxs, err := store.ImportNDJSON(f)
		if err != nil {
			return fmt.Errorf("streaming ndjson import: %w", err)
		}
		fmt.Printf("Streamed %d commands, %d contexts from %s\n", importedCmds, importedCtxs, importInput)
		return nil
	}

	fileData, err := os.ReadFile(importInput)
	if err != nil {
		return fmt.Errorf("reading import file: %w", err)
	}

	var jsonData []byte

	// Detect format: try to parse as RECL encrypted format
	salt, encryptedData, unpackErr := vault.UnpackExport(fileData)
	if unpackErr == nil {
		// Encrypted file
		password := importPassword
		if password == "" {
			password, err = vault.ReadPassword("Enter decryption password: ")
			if err != nil {
				return fmt.Errorf("password input: %w", err)
			}
		}

		key, dkErr := vault.DeriveKey(password, salt)
		if dkErr != nil {
			return fmt.Errorf("deriving decryption key: %w", dkErr)
		}
		jsonData, err = vault.Decrypt(encryptedData, key)
		if err != nil {
			return fmt.Errorf("decrypting import: %w", err)
		}
	} else {
		// Try as plain JSON
		jsonData = fileData
	}

	var exportData vault.ExportData
	if err := json.Unmarshal(jsonData, &exportData); err != nil {
		return fmt.Errorf("parsing import data: %w", err)
	}

	action := "Replaced"
	if importMerge {
		action = "Merged"
	}

	importedCmds, importedCtxs, err := store.ImportVaultData(&exportData, importMerge)
	if err != nil {
		return fmt.Errorf("importing data: %w", err)
	}

	fmt.Printf("%s %d commands, %d contexts from %s\n",
		action, importedCmds, importedCtxs, importInput)

	return nil
}

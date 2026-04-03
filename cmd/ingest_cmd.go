package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/Om-Rohilla/recall/internal/capture"
	"github.com/Om-Rohilla/recall/internal/ui"
	"github.com/Om-Rohilla/recall/pkg/config"
	"github.com/spf13/cobra"
)

var ingestCmd = &cobra.Command{
	Use:    "ingest",
	Hidden: true,
	Short:  "Internal command. Processes the background capture queue.",
	RunE:   runIngest,
}

func init() {
	rootCmd.AddCommand(ingestCmd)
}

func runIngest(cmd *cobra.Command, args []string) error {
	cfg := config.Get()
	if !cfg.Capture.Enabled {
		return nil
	}

	queueFile := filepath.Join(filepath.Dir(cfg.Vault.Path), "pending.ndjson")
	if _, err := os.Stat(queueFile); os.IsNotExist(err) {
		return nil
	}

	// This is running completely detached from the user's terminal prompt via exec.Command
	err := capture.IngestQueue(queueFile, cfg)
	if err != nil {
		return fmt.Errorf("ingestion failed: %w", err)
	}

	// In debug mode, we could log something, but hidden commands should remain silent
	if os.Getenv("RECALL_DEBUG") == "1" {
		fmt.Println(ui.DimStyle.Render("Ingest job completed."))
	}

	return nil
}

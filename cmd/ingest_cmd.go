package cmd

import (
	"fmt"
	"os"

	"github.com/Om-Rohilla/recall/internal/capture"
	"github.com/Om-Rohilla/recall/internal/ui"
	"github.com/Om-Rohilla/recall/pkg/config"
	"github.com/spf13/cobra"
)

var ingestFile string

var ingestCmd = &cobra.Command{
	Use:    "ingest",
	Hidden: true,
	Short:  "Internal command. Processes the background capture queue.",
	RunE:   runIngest,
}

func init() {
	ingestCmd.Flags().StringVar(&ingestFile, "file", "", "specific queue file to ingest (internal use)")
	rootCmd.AddCommand(ingestCmd)
}

func runIngest(cmd *cobra.Command, args []string) error {
	cfg := config.Get()
	if !cfg.Capture.Enabled {
		return nil
	}

	// IngestQueue accepts "" to use the default pending.ndjson path (backward compat).
	err := capture.IngestQueue(ingestFile, cfg)
	if err != nil {
		return fmt.Errorf("ingestion failed: %w", err)
	}

	// In debug mode, we could log something, but hidden commands should remain silent
	if os.Getenv("RECALL_DEBUG") == "1" {
		fmt.Println(ui.DimStyle.Render("Ingest job completed."))
	}

	return nil
}

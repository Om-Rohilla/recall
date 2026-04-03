package cmd

import (
	"os"
	"os/exec"
	"strconv"
	"time"

	"github.com/Om-Rohilla/recall/internal/capture"
	"github.com/Om-Rohilla/recall/internal/vault"
	"github.com/Om-Rohilla/recall/pkg/config"
	"github.com/spf13/cobra"
)

var (
	captureCommand  string
	captureExitCode string
	captureCwd      string
	captureDuration string
	captureSession  string
)

var captureCmd = &cobra.Command{
	Use:    "capture",
	Short:  "Capture a command (called by shell hook)",
	Hidden: true,
	RunE:   runCapture,
}

func init() {
	captureCmd.Flags().StringVar(&captureCommand, "command", "", "the command that was executed")
	captureCmd.Flags().StringVar(&captureExitCode, "exit-code", "0", "exit code of the command")
	captureCmd.Flags().StringVar(&captureCwd, "cwd", "", "working directory")
	captureCmd.Flags().StringVar(&captureDuration, "duration", "0", "duration in milliseconds")
	captureCmd.Flags().StringVar(&captureSession, "session", "", "session ID")
	rootCmd.AddCommand(captureCmd)
}

func runCapture(cmd *cobra.Command, args []string) error {
	if captureCommand == "" {
		return nil
	}

	if os.Getenv("RECALL_DAEMONIZED") == "" {
		os.Setenv("RECALL_DAEMONIZED", "1")
		daemonCmd := exec.Command(os.Args[0], os.Args[1:]...)
		_ = daemonCmd.Start()
		return nil
	}

	cfg := config.Get()
	exitCode, _ := strconv.Atoi(captureExitCode)
	durationMs, _ := strconv.Atoi(captureDuration)

	data := &vault.CaptureData{
		RawCommand: captureCommand,
		ExitCode:   exitCode,
		Cwd:        captureCwd,
		Timestamp:  time.Now().UTC(),
		DurationMs: durationMs,
		SessionID:  captureSession,
	}

	return capture.AppendAndTryFlush(data, cfg)
}

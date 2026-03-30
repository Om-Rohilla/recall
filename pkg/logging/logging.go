package logging

import (
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"sync"
)

var (
	logger *slog.Logger
	once   sync.Once
)

// Init initializes the structured logger.
// If RECALL_DEBUG=1, logs at Debug level to a file; otherwise discards.
// Always writes to ~/.local/share/recall/recall.log when the file is accessible.
func Init() *slog.Logger {
	once.Do(func() {
		debug := os.Getenv("RECALL_DEBUG") == "1"

		var w io.Writer = io.Discard
		if debug {
			logPath := logFilePath()
			dir := filepath.Dir(logPath)
			if err := os.MkdirAll(dir, 0o700); err == nil {
				if f, err := os.OpenFile(logPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600); err == nil {
					w = f
				}
			}
		}

		level := slog.LevelInfo
		if debug {
			level = slog.LevelDebug
		}

		logger = slog.New(slog.NewJSONHandler(w, &slog.HandlerOptions{
			Level: level,
		}))
	})
	return logger
}

// Get returns the initialized logger (or a discard logger if Init hasn't been called).
func Get() *slog.Logger {
	if logger == nil {
		return Init()
	}
	return logger
}

func logFilePath() string {
	if v := os.Getenv("RECALL_LOG"); v != "" {
		return v
	}
	dataDir := os.Getenv("XDG_DATA_HOME")
	if dataDir == "" {
		home, err := os.UserHomeDir()
		if err != nil || home == "" {
			home = os.Getenv("HOME")
		}
		if home == "" {
			home = "/tmp"
		}
		dataDir = filepath.Join(home, ".local", "share")
	}
	return filepath.Join(dataDir, "recall", "recall.log")
}

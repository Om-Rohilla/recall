package logging

import (
	"fmt"
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

const (
	// DefaultMaxLogSize is the maximum log file size before rotation (5 MB).
	DefaultMaxLogSize int64 = 5 * 1024 * 1024
	// DefaultKeepFiles is the number of rotated log files to keep.
	DefaultKeepFiles = 3
)

// Init initializes the structured logger.
// Always writes INFO-level logs to ~/.local/share/recall/recall.log.
// If RECALL_DEBUG=1, logs at Debug level for verbose output.
// Log rotation is performed automatically if the log file exceeds DefaultMaxLogSize.
func Init() *slog.Logger {
	once.Do(func() {
		debug := os.Getenv("RECALL_DEBUG") == "1"

		// Always log to file at INFO level for production observability.
		// Debug mode increases verbosity.
		var w io.Writer = io.Discard
		logPath := logFilePath()
		dir := filepath.Dir(logPath)
		if err := os.MkdirAll(dir, 0o700); err == nil {
			// Rotate before opening
			rotateLogFile(logPath, DefaultMaxLogSize, DefaultKeepFiles)

			if f, err := os.OpenFile(logPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600); err == nil {
				w = f
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

// RotateLogs forces log rotation regardless of file size.
func RotateLogs() error {
	logPath := logFilePath()
	info, err := os.Stat(logPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("stat log file: %w", err)
	}
	if info.Size() == 0 {
		return nil
	}
	return doRotate(logPath, DefaultKeepFiles)
}

// rotateLogFile checks the log file size and rotates if it exceeds maxSize.
func rotateLogFile(path string, maxSize int64, keepN int) {
	info, err := os.Stat(path)
	if err != nil {
		return // File doesn't exist yet, no rotation needed
	}
	if info.Size() < maxSize {
		return
	}
	_ = doRotate(path, keepN)
}

// doRotate performs the actual file rotation: .3 → delete, .2 → .3, .1 → .2, current → .1
func doRotate(path string, keepN int) error {
	// Remove the oldest rotated file
	oldest := fmt.Sprintf("%s.%d", path, keepN)
	os.Remove(oldest)

	// Shift existing rotated files
	for i := keepN - 1; i >= 1; i-- {
		src := fmt.Sprintf("%s.%d", path, i)
		dst := fmt.Sprintf("%s.%d", path, i+1)
		os.Rename(src, dst)
	}

	// Move current log to .1
	if err := os.Rename(path, path+".1"); err != nil {
		return fmt.Errorf("rotating log file: %w", err)
	}

	return nil
}

// LogFilePath returns the current log file path (exported for maintenance command).
func LogFilePath() string {
	return logFilePath()
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

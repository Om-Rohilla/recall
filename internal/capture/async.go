package capture

import (
	"bufio"
	"bytes"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"syscall"

	"github.com/Om-Rohilla/recall/internal/vault"
	"github.com/Om-Rohilla/recall/pkg/config"
	"github.com/Om-Rohilla/recall/pkg/logging"
)

// AppendAndTryFlush appends the current capture data to a wait-free append-only log.
// It then attempts a non-blocking lock to ingest the log into SQLite.
// If the lock is held by another concurrent prompt, it exits instantly (0 blocking).
func AppendAndTryFlush(data *vault.CaptureData, cfg *config.Config) error {
	log := logging.Get()
	if !cfg.Capture.Enabled {
		return nil
	}

	queueFile := filepath.Join(filepath.Dir(cfg.Vault.Path), "pending.ndjson")
	
	// 1. Thread-safe atomic append
	f, err := os.OpenFile(queueFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)
	if err != nil {
		return err
	}
	
	b, err := json.Marshal(data)
	if err != nil {
		f.Close()
		return err
	}
	b = append(b, '\n')
	if _, err := f.Write(b); err != nil {
		f.Close()
		return err
	}
	f.Close()

	// 2. Try to flush the queue
	lockFile := filepath.Join(filepath.Dir(cfg.Vault.Path), "flush.lock")
	lf, err := os.OpenFile(lockFile, os.O_CREATE|os.O_RDWR, 0600)
	if err != nil {
		return err
	}
	defer lf.Close()

	// Attempt non-blocking exclusive lock
	if err := syscall.Flock(int(lf.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
		// Locked by another prompt. Let them handle the ingest! Zero-latency exit.
		log.Debug("flush locked by another process, exiting cleanly")
		return nil
	}
	defer syscall.Flock(int(lf.Fd()), syscall.LOCK_UN)

	// We got the lock. Ingest everything in the queue.
	return ingestQueue(queueFile, cfg)
}

func ingestQueue(queueFile string, cfg *config.Config) error {
	qf, err := os.OpenFile(queueFile, os.O_RDWR, 0600)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	
	b, err := io.ReadAll(qf)
	if err != nil {
		qf.Close()
		return err
	}
	
	// Truncate immediately so incoming parallel hooks start fresh
	if err := qf.Truncate(0); err != nil {
		qf.Close()
		return err
	}
	qf.Close()

	if len(bytes.TrimSpace(b)) == 0 {
		return nil
	}

	// Open the SQLite database since we have data to ingest
	store, err := vault.NewStore(cfg.Vault.Path)
	if err != nil {
		return err
	}
	defer store.Close()

	scanner := bufio.NewScanner(bytes.NewReader(b))
	var lastErr error
	for scanner.Scan() {
		line := bytes.TrimSpace(scanner.Bytes())
		if len(line) == 0 {
			continue
		}
		var cd vault.CaptureData
		if err := json.Unmarshal(line, &cd); err == nil {
			// Process each command fully (Parse, Filter, Enrich, Store)
			if err := ProcessCommand(store, &cd, cfg); err != nil {
				lastErr = err
			}
		}
	}
	
	return lastErr
}

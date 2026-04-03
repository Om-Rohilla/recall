package capture

import (
	"bufio"
	"bytes"
	"encoding/base64"
	"encoding/json"
	"io"
	"os"
	"os/exec"
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
	
	key, err := vault.GetOrGenerateVaultKey()
	if err != nil {
		f.Close()
		return err
	}
	encData, err := vault.Encrypt(b, key)
	if err != nil {
		f.Close()
		return err
	}
	
	bEnc := make([]byte, base64.StdEncoding.EncodedLen(len(encData)))
	base64.StdEncoding.Encode(bEnc, encData)
	bEnc = append(bEnc, '\n')
	if _, err := f.Write(bEnc); err != nil {
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

	// We got the lock. This means no background ingest is currently running.
	// We MUST release the lock so the background process can grab it.
	// The background process will ingest everything in the queue.
	if err := syscall.Flock(int(lf.Fd()), syscall.LOCK_UN); err != nil {
		log.Debug("failed to unlock flush lock", "error", err)
	}

	executable, err := os.Executable()
	if err != nil {
		return err
	}

	// Spawn completely detached
	cmd := exec.Command(executable, "ingest")
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Setpgid: true, // Prevent sending signals from terminal to this process
	}
	
	if err := cmd.Start(); err != nil {
		return err
	}

	log.Debug("spawned background ingest process", "pid", cmd.Process.Pid)
	return nil
}

func IngestQueue(queueFile string, cfg *config.Config) error {
	// Let the background daemon cleanly acquire the lock first
	lockFile := filepath.Join(filepath.Dir(cfg.Vault.Path), "flush.lock")
	lf, err := os.OpenFile(lockFile, os.O_CREATE|os.O_RDWR, 0600)
	if err != nil {
		return err
	}
	defer lf.Close()

	if err := syscall.Flock(int(lf.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
		return nil // someone else got the lock
	}
	defer syscall.Flock(int(lf.Fd()), syscall.LOCK_UN)

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
	key, err := vault.GetOrGenerateVaultKey()
	if err != nil {
		return err
	}

	for scanner.Scan() {
		line := bytes.TrimSpace(scanner.Bytes())
		if len(line) == 0 {
			continue
		}
		
		decLen := base64.StdEncoding.DecodedLen(len(line))
		decoded := make([]byte, decLen)
		n, err := base64.StdEncoding.Decode(decoded, line)
		if err != nil {
			continue
		}
		
		plain, err := vault.Decrypt(decoded[:n], key)
		if err != nil {
			continue
		}
		
		var cd vault.CaptureData
		if err := json.Unmarshal(plain, &cd); err == nil {
			// Process each command fully (Parse, Filter, Enrich, Store)
			if err := ProcessCommand(store, &cd, cfg); err != nil {
				lastErr = err
			}
		}
	}
	
	return lastErr
}

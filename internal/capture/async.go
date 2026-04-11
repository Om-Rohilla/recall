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
	"strconv"
	"syscall"

	"github.com/Om-Rohilla/recall/internal/vault"
	"github.com/Om-Rohilla/recall/pkg/config"
	"github.com/Om-Rohilla/recall/pkg/logging"
)

// AppendAndTryFlush appends the current capture data to a wait-free append-only log.
// It then attempts to atomically claim the queue via file rename and spawn a detached
// background ingest process. Only one ingest process can ever claim a given queue file,
// eliminating the race between flock release and process spawn.
func AppendAndTryFlush(data *vault.CaptureData, cfg *config.Config) error {
	log := logging.Get()
	if !cfg.Capture.Enabled {
		return nil
	}

	queueFile := filepath.Join(filepath.Dir(cfg.Vault.Path), "pending.ndjson")

	// 1. Thread-safe atomic append (O_APPEND writes < PIPE_BUF are atomic on POSIX)
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

	// 2. Acquire a non-blocking exclusive lock before attempting the atomic rename.
	// This serialises the rename step so exactly one process wins per flush cycle.
	lockFile := filepath.Join(filepath.Dir(cfg.Vault.Path), "flush.lock")
	lf, err := os.OpenFile(lockFile, os.O_CREATE|os.O_RDWR, 0600)
	if err != nil {
		return err
	}
	defer lf.Close()

	if err := syscall.Flock(int(lf.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
		// Locked by another prompt. Let them handle the ingest! Zero-latency exit.
		log.Debug("flush locked by another process, exiting cleanly")
		return nil
	}

	// Atomically claim the queue by renaming it to a pid-unique ingest file.
	// New appends from concurrent hooks will create a fresh pending.ndjson.
	ingestFile := queueFile + "." + strconv.Itoa(os.Getpid())
	if err := os.Rename(queueFile, ingestFile); err != nil {
		// Queue file absent or another process just claimed it — exit cleanly.
		_ = syscall.Flock(int(lf.Fd()), syscall.LOCK_UN)
		return nil
	}

	// Release the lock NOW so new writers can start appending to a fresh queue.
	_ = syscall.Flock(int(lf.Fd()), syscall.LOCK_UN)

	executable, err := os.Executable()
	if err != nil {
		return err
	}

	// Spawn completely detached, passing the specific ingest file.
	cmd := exec.Command(executable, "ingest", "--file", ingestFile)
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Setpgid: true, // Prevent signals from terminal reaching this process
	}

	if err := cmd.Start(); err != nil {
		return err
	}

	log.Debug("spawned background ingest process", "pid", cmd.Process.Pid, "file", ingestFile)
	return nil
}

// IngestQueue processes the captured command queue from the given file path.
// When file is empty-string it falls back to the default pending.ndjson location
// (for backward compatibility). After ingest the file is removed.
func IngestQueue(file string, cfg *config.Config) error {
	if file == "" {
		file = filepath.Join(filepath.Dir(cfg.Vault.Path), "pending.ndjson")
	}

	qf, err := os.Open(file)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	b, err := io.ReadAll(qf)
	qf.Close()
	if err != nil {
		return err
	}

	// Remove the claimed ingest file immediately so it doesn't get reprocessed.
	_ = os.Remove(file)

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

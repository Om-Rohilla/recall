package plugin

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/Om-Rohilla/recall/pkg/logging"
	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/imports/wasi_snapshot_preview1"
)

// Plugin represents a WASM executable module that extends Recall securely.
type Plugin struct {
	Name    string   `json:"name"`
	Path    string   `json:"path"`
	Version string   `json:"version"`
	Hooks   []string `json:"hooks"`
}

// Registry manages installed plugins.
type Registry struct {
	dir string
}

// NewRegistry initializes a plugin registry pointing to the user's plugin directory.
func NewRegistry() *Registry {
	dataDir := os.Getenv("XDG_DATA_HOME")
	if dataDir == "" {
		home, _ := os.UserHomeDir()
		if home == "" {
			home = os.Getenv("HOME")
		}
		if home == "" {
			home = "/tmp"
		}
		dataDir = filepath.Join(home, ".local", "share")
	}
	dir := filepath.Join(dataDir, "recall", "plugins")
	_ = os.MkdirAll(dir, 0o700)
	return &Registry{dir: dir}
}

// Dir returns the plugin directory path.
func (r *Registry) Dir() string {
	return r.dir
}

// List discovers and queries all installed WASM plugins.
func (r *Registry) List() ([]Plugin, error) {
	entries, err := os.ReadDir(r.dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("reading plugin directory: %w", err)
	}

	var plugins []Plugin
	for _, entry := range entries {
		if entry.IsDir() || strings.HasPrefix(entry.Name(), ".") {
			continue
		}

		// Security: Only execute strict .wasm modules
		if !strings.HasSuffix(entry.Name(), ".wasm") {
			continue
		}

		path := filepath.Join(r.dir, entry.Name())
		p := r.queryPlugin(path, entry.Name())
		plugins = append(plugins, p)
	}

	return plugins, nil
}

// queryPlugin asks the WASM module for its metadata via the '--recall-plugin-info' flag.
// Uses `wazero` to securely sandbox the plugin. It has ZERO system access except stdout.
// A 10-second timeout prevents a malicious or looping plugin from hanging the process.
func (r *Registry) queryPlugin(path string, fallbackName string) Plugin {
	p := Plugin{Name: fallbackName, Path: path, Version: "unknown", Hooks: []string{}}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	wasmBytes, err := os.ReadFile(path)
	if err != nil {
		logging.Get().Debug("failed to read wasm plugin", "path", path, "error", err)
		return p
	}

	runtime := wazero.NewRuntime(ctx)
	defer runtime.Close(ctx)

	// Instantiate WASI preview 1, which provides standard out/err/in capabilities.
	wasi_snapshot_preview1.MustInstantiate(ctx, runtime)

	var stdoutBuf bytes.Buffer
	config := wazero.NewModuleConfig().
		WithName(fallbackName).
		WithArgs(fallbackName, "--recall-plugin-info").
		WithStdout(&stdoutBuf) // Securely capture only what the plugin prints
		// Notice: WithFS, WithEnv, WithNet are entirely omitted, guaranteeing isolation.

	_, err = runtime.InstantiateWithConfig(ctx, wasmBytes, config)
	if err != nil {
		// WASI exit code != 0 are returned as errors, but if stdoutBuf has our data, we don't care.
		logging.Get().Debug("wasm instantiation completed with note", "err", err)
	}

	var metadata struct {
		Name    string   `json:"name"`
		Version string   `json:"version"`
		Hooks   []string `json:"hooks"`
	}
	if err := json.Unmarshal(stdoutBuf.Bytes(), &metadata); err == nil {
		if metadata.Name != "" {
			p.Name = metadata.Name
		}
		if metadata.Version != "" {
			p.Version = metadata.Version
		}
		p.Hooks = metadata.Hooks
	} else {
		logging.Get().Debug("plugin JSON parse failed", "output", stdoutBuf.String())
	}
	
	return p
}

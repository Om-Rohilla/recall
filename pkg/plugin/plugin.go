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
	"github.com/tetratelabs/wazero/api"
)

// wasmMagic is the 4-byte module header that all valid .wasm files begin with.
const wasmMagic = "\x00asm"

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

// isValidWASM checks the magic bytes to confirm a file is a WASM module.
// Prevents non-WASM executables from being treated as plugins.
func isValidWASM(data []byte) bool {
	return len(data) >= 4 && string(data[:4]) == wasmMagic
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

		// Security: Validate WASM magic bytes before executing.
		// A non-WASM file in the plugin directory (e.g. dropped by an attacker)
		// must not be instantiated — wazero would panic on malformed input.
		data, err := os.ReadFile(path)
		if err != nil || !isValidWASM(data) {
			logging.Get().Warn("skipping invalid or unreadable WASM file", "path", path)
			continue
		}

		p := r.queryPlugin(path, entry.Name(), data)
		plugins = append(plugins, p)
	}

	return plugins, nil
}

// queryPlugin asks the WASM module for its metadata via the '--recall-plugin-info' flag.
// Uses a minimal custom WASI host module to sandbox the plugin. The sandbox provides
// ONLY fd_write (stdout/stderr), proc_exit, and ENOSYS stubs for all other WASI calls.
// This replaces the blanket wasi_snapshot_preview1.MustInstantiate which exposed
// filesystem, environment, clocks, and random access — far more syscall surface than needed.
//
// A 10-second timeout prevents a malicious or looping plugin from hanging the process.
func (r *Registry) queryPlugin(path string, fallbackName string, wasmBytes []byte) Plugin {
	p := Plugin{Name: fallbackName, Path: path, Version: "unknown", Hooks: []string{}}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	runtime := wazero.NewRuntimeWithConfig(ctx,
		wazero.NewRuntimeConfig().WithCloseOnContextDone(true))
	defer runtime.Close(ctx)

	var stdoutBuf bytes.Buffer

	// Minimal WASI host module: expose only what a metadata query plugin needs.
	// All other WASI imports return ENOSYS (52) so the module can compile but
	// cannot perform filesystem I/O, environment reads, network calls, etc.
	_, err := runtime.NewHostModuleBuilder("wasi_snapshot_preview1").
		NewFunctionBuilder().
		WithFunc(func(ctx context.Context, m api.Module, fd, iovs, iovsCount, resultNwritten uint32) uint32 {
			// fd_write: only allow writes to stdout (fd=1) and stderr (fd=2)
			if fd != 1 && fd != 2 {
				return 8 // EBADF
			}
			size := uint32(0)
			for i := uint32(0); i < iovsCount; i++ {
				iov := iovs + i*8
				offset, _ := m.Memory().ReadUint32Le(iov)
				length, _ := m.Memory().ReadUint32Le(iov + 4)
				if data, ok := m.Memory().Read(offset, length); ok {
					if fd == 1 {
						stdoutBuf.Write(data)
					}
					size += uint32(len(data))
				}
			}
			m.Memory().WriteUint32Le(resultNwritten, size)
			return 0 // ESUCCESS
		}).Export("fd_write").
		NewFunctionBuilder().
		WithFunc(func(ctx context.Context, code uint32) {
			// proc_exit: context cancellation handles cleanup
		}).Export("proc_exit").
		// Stub all remaining required WASI imports as ENOSYS (no-ops that prevent the
		// plugin from successfully performing privileged operations).
		NewFunctionBuilder().WithFunc(func() uint32 { return 52 }).Export("fd_read").
		NewFunctionBuilder().WithFunc(func() uint32 { return 52 }).Export("random_get").
		NewFunctionBuilder().WithFunc(func() uint32 { return 52 }).Export("environ_get").
		NewFunctionBuilder().WithFunc(func() uint32 { return 52 }).Export("environ_sizes_get").
		NewFunctionBuilder().WithFunc(func() uint32 { return 52 }).Export("clock_time_get").
		NewFunctionBuilder().WithFunc(func() uint32 { return 52 }).Export("path_open").
		NewFunctionBuilder().WithFunc(func() uint32 { return 52 }).Export("fd_close").
		NewFunctionBuilder().WithFunc(func() uint32 { return 52 }).Export("fd_seek").
		NewFunctionBuilder().WithFunc(func() uint32 { return 52 }).Export("fd_fdstat_get").
		NewFunctionBuilder().WithFunc(func() uint32 { return 52 }).Export("fd_prestat_get").
		NewFunctionBuilder().WithFunc(func() uint32 { return 52 }).Export("fd_prestat_dir_name").
		Instantiate(ctx)
	if err != nil {
		logging.Get().Debug("WASI host module instantiation failed", "path", path, "error", err)
		return p
	}

	config := wazero.NewModuleConfig().
		WithName(fallbackName).
		WithArgs(fallbackName, "--recall-plugin-info").
		WithStdout(&stdoutBuf)
		// WithFS, WithEnv, WithNet intentionally omitted — no filesystem/env/net access.

	_, err = runtime.InstantiateWithConfig(ctx, wasmBytes, config)
	if err != nil {
		// WASI exit code != 0 is returned as an error, but if stdoutBuf has our data, that's fine.
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

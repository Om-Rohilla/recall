package plugin

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/Om-Rohilla/recall/pkg/logging"
)

// Plugin represents an executable that extends Recall.
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

// List discovers and queries all installed plugins.
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

		path := filepath.Join(r.dir, entry.Name())
		info, err := entry.Info()
		if err != nil || info.Mode()&0111 == 0 {
			continue // Not executable
		}

		p := r.queryPlugin(path, entry.Name())
		plugins = append(plugins, p)
	}

	return plugins, nil
}

// queryPlugin asks the executable for its metadata via the '--recall-plugin-info' flag.
func (r *Registry) queryPlugin(path string, fallbackName string) Plugin {
	p := Plugin{Name: fallbackName, Path: path, Version: "unknown", Hooks: []string{}}
	
	cmd := exec.Command(path, "--recall-plugin-info")
	out, err := cmd.Output()
	if err != nil {
		logging.Get().Debug("plugin failed to respond to info query", "path", path, "error", err)
		return p // Return basic info if query fails
	}
	
	// Try parsing JSON output
	var metadata struct {
		Name    string   `json:"name"`
		Version string   `json:"version"`
		Hooks   []string `json:"hooks"`
	}
	if err := json.Unmarshal(out, &metadata); err == nil {
		if metadata.Name != "" {
			p.Name = metadata.Name
		}
		if metadata.Version != "" {
			p.Version = metadata.Version
		}
		p.Hooks = metadata.Hooks
	}
	
	return p
}

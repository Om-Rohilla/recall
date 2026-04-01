package intelligence

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/Om-Rohilla/recall/internal/vault"
)

type KnowledgeEntry struct {
	Command     string   `json:"command"`
	Description string   `json:"description"`
	Intents     []string `json:"intents"`
	Category    string   `json:"category"`
	DangerLevel string   `json:"danger_level,omitempty"`
}

// KBCache stores the hash of the last-loaded knowledge base file.
type KBCache struct {
	Hash       string `json:"hash"`
	EntryCount int    `json:"entry_count"`
}

func LoadKnowledgeBase(store *vault.Store, jsonPath string) (int, error) {
	// Compute current file hash
	hash, err := fileHash(jsonPath)
	if err != nil {
		return 0, fmt.Errorf("hashing knowledge base file: %w", err)
	}

	// Check cache
	cacheDir := kbCacheDir()
	cache, _ := loadKBCache(cacheDir)

	count, countErr := store.KnowledgeCount()
	if countErr != nil {
		return 0, fmt.Errorf("checking knowledge count: %w", countErr)
	}

	// If hash matches and data already loaded, skip
	if cache != nil && cache.Hash == hash && count > 0 {
		return count, nil
	}

	data, err := os.ReadFile(jsonPath)
	if err != nil {
		return 0, fmt.Errorf("reading knowledge base file %s: %w", jsonPath, err)
	}

	var entries []KnowledgeEntry
	if err := json.Unmarshal(data, &entries); err != nil {
		return 0, fmt.Errorf("parsing knowledge base JSON: %w", err)
	}

	var knowledgeRows []vault.Knowledge
	for _, e := range entries {
		intentsJSON, _ := json.Marshal(e.Intents)
		dangerLevel := e.DangerLevel
		if dangerLevel == "" {
			dangerLevel = "safe"
		}
		knowledgeRows = append(knowledgeRows, vault.Knowledge{
			Command:     e.Command,
			Description: e.Description,
			Intents:     string(intentsJSON),
			Category:    e.Category,
			FlagsDoc:    "{}",
			Examples:    "[]",
			DangerLevel: dangerLevel,
		})
	}

	inserted, err := store.BatchInsertKnowledge(knowledgeRows)
	if err != nil {
		return 0, fmt.Errorf("inserting knowledge base: %w", err)
	}

	if err := store.RebuildKnowledgeFTSIndex(); err != nil {
		return inserted, fmt.Errorf("rebuilding knowledge FTS index: %w", err)
	}

	// Save cache
	newCache := &KBCache{
		Hash:       hash,
		EntryCount: inserted,
	}
	_ = saveKBCache(cacheDir, newCache)

	return inserted, nil
}

func FindKnowledgeBasePath() string {
	candidates := []string{
		"data/knowledge.json",
		"/usr/local/share/recall/knowledge.json",
		"/usr/share/recall/knowledge.json",
	}

	execPath, err := os.Executable()
	if err == nil {
		dir := filepath.Dir(execPath)
		for i := 0; i < 3; i++ {
			candidates = append(candidates, filepath.Join(dir, "data", "knowledge.json"))
			parent := filepath.Dir(dir)
			if parent == dir {
				break
			}
			dir = parent
		}
	}

	homeDir, _ := os.UserHomeDir()
	if homeDir != "" {
		candidates = append(candidates, filepath.Join(homeDir, ".local", "share", "recall", "knowledge.json"))
	}

	for _, path := range candidates {
		if _, err := os.Stat(path); err == nil {
			return path
		}
	}
	return ""
}

// fileHash computes the SHA256 hash of a file and returns it as a hex string.
func fileHash(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:]), nil
}

// kbCacheDir returns the directory to store the KB cache file.
func kbCacheDir() string {
	configDir := os.Getenv("XDG_CONFIG_HOME")
	if configDir == "" {
		home, _ := os.UserHomeDir()
		if home == "" {
			home = os.Getenv("HOME")
		}
		if home == "" {
			home = "/tmp"
		}
		configDir = filepath.Join(home, ".config")
	}
	return filepath.Join(configDir, "recall")
}

// loadKBCache reads the cached KB hash from disk.
func loadKBCache(dir string) (*KBCache, error) {
	path := filepath.Join(dir, "kb_cache.json")
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var cache KBCache
	if err := json.Unmarshal(data, &cache); err != nil {
		return nil, err
	}
	return &cache, nil
}

// saveKBCache writes the KB hash to disk.
func saveKBCache(dir string, cache *KBCache) error {
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	data, err := json.Marshal(cache)
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, "kb_cache.json"), data, 0o600)
}

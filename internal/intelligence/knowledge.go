package intelligence

import (
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

func LoadKnowledgeBase(store *vault.Store, jsonPath string) (int, error) {
	count, err := store.KnowledgeCount()
	if err != nil {
		return 0, fmt.Errorf("checking knowledge count: %w", err)
	}
	if count > 0 {
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

package tests

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/Om-Rohilla/recall/cmd"
)

type KBEntry struct {
	Command     string   `json:"command"`
	Description string   `json:"description"`
	Intents     []string `json:"intents"`
	Category    string   `json:"category"`
	DangerLevel string   `json:"danger_level"`
}

func loadKnowledgeBase(t *testing.T) []KBEntry {
	t.Helper()
	data, err := os.ReadFile("../data/knowledge.json")
	if err != nil {
		t.Fatalf("reading knowledge.json: %v", err)
	}
	var entries []KBEntry
	if err := json.Unmarshal(data, &entries); err != nil {
		t.Fatalf("parsing knowledge.json: %v", err)
	}
	return entries
}

func TestVersionAndBuildDateVarsExist(t *testing.T) {
	if cmd.Version == "" {
		t.Fatal("cmd.Version should not be empty")
	}
	if cmd.BuildDate == "" {
		t.Fatal("cmd.BuildDate should not be empty")
	}
	// Default values are "dev" and "unknown" — both are non-empty
	t.Logf("Version=%s BuildDate=%s", cmd.Version, cmd.BuildDate)
}

func TestGoReleaserConfigExists(t *testing.T) {
	info, err := os.Stat("../.goreleaser.yml")
	if err != nil {
		t.Fatalf(".goreleaser.yml not found: %v", err)
	}
	if info.Size() == 0 {
		t.Fatal(".goreleaser.yml is empty")
	}
}

func TestInstallScriptExists(t *testing.T) {
	data, err := os.ReadFile("../scripts/install.sh")
	if err != nil {
		t.Fatalf("scripts/install.sh not found: %v", err)
	}
	if len(data) < 10 {
		t.Fatal("install.sh is too short")
	}
	if string(data[:2]) != "#!" {
		t.Fatal("install.sh should start with a shebang (#!)")
	}
}

func TestKnowledgeBaseMinEntries(t *testing.T) {
	entries := loadKnowledgeBase(t)
	if len(entries) < 500 {
		t.Fatalf("knowledge base has %d entries, want at least 500", len(entries))
	}
	t.Logf("knowledge base has %d entries", len(entries))
}

func TestKnowledgeBaseNoDuplicates(t *testing.T) {
	entries := loadKnowledgeBase(t)
	seen := make(map[string]int)
	for i, e := range entries {
		if prev, ok := seen[e.Command]; ok {
			t.Errorf("duplicate command at index %d and %d: %q", prev, i, e.Command)
		}
		seen[e.Command] = i
	}
}

func TestKnowledgeBaseDangerLevels(t *testing.T) {
	entries := loadKnowledgeBase(t)
	valid := map[string]bool{"safe": true, "caution": true, "destructive": true}
	for i, e := range entries {
		if !valid[e.DangerLevel] {
			t.Errorf("entry %d (%q): invalid danger_level %q", i, e.Command, e.DangerLevel)
		}
	}
}

func TestKnowledgeBaseCategories(t *testing.T) {
	entries := loadKnowledgeBase(t)
	for i, e := range entries {
		if e.Category == "" {
			t.Errorf("entry %d (%q): empty category", i, e.Command)
		}
	}
}

func TestKnowledgeBaseIntents(t *testing.T) {
	entries := loadKnowledgeBase(t)
	for i, e := range entries {
		if len(e.Intents) < 2 {
			t.Errorf("entry %d (%q): only %d intents, want at least 2", i, e.Command, len(e.Intents))
		}
	}
}

func TestKnowledgeBaseRequiredFields(t *testing.T) {
	entries := loadKnowledgeBase(t)
	for i, e := range entries {
		if e.Command == "" {
			t.Errorf("entry %d: empty command", i)
		}
		if e.Description == "" {
			t.Errorf("entry %d (%q): empty description", i, e.Command)
		}
		if len(e.Intents) == 0 {
			t.Errorf("entry %d (%q): empty intents", i, e.Command)
		}
	}
}

func TestShellHookScriptsExist(t *testing.T) {
	hooks := []string{
		"../scripts/hooks/recall.zsh",
		"../scripts/hooks/recall.bash",
		"../scripts/hooks/recall.fish",
	}
	for _, path := range hooks {
		info, err := os.Stat(path)
		if err != nil {
			t.Errorf("hook script not found: %s", path)
			continue
		}
		if info.Size() == 0 {
			t.Errorf("hook script is empty: %s", path)
		}
	}
}

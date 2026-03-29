package tests

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/Om-Rohilla/recall/internal/ui"
	"github.com/Om-Rohilla/recall/internal/vault"
)

func setupTestVault(t *testing.T) (*vault.Store, func()) {
	t.Helper()
	tmpDir, err := os.MkdirTemp("", "recall-ui-test-*")
	if err != nil {
		t.Fatal(err)
	}
	dbPath := filepath.Join(tmpDir, "test.db")
	store, err := vault.NewStore(dbPath)
	if err != nil {
		os.RemoveAll(tmpDir)
		t.Fatal(err)
	}
	return store, func() {
		store.Close()
		os.RemoveAll(tmpDir)
	}
}

func seedVault(t *testing.T, store *vault.Store) {
	t.Helper()
	now := time.Now().UTC()
	cmds := []vault.Command{
		{Raw: "git commit -m 'test'", Binary: "git", Subcommand: "commit", Flags: "[]", Category: "git", Frequency: 47, FirstSeen: now.Add(-30 * 24 * time.Hour), LastSeen: now.Add(-1 * time.Hour)},
		{Raw: "docker compose up -d", Binary: "docker", Subcommand: "compose up", Flags: `["-d"]`, Category: "docker", Frequency: 23, FirstSeen: now.Add(-20 * 24 * time.Hour), LastSeen: now.Add(-2 * time.Hour)},
		{Raw: "kubectl get pods -n staging", Binary: "kubectl", Subcommand: "get pods", Flags: `["-n"]`, Category: "kubernetes", Frequency: 19, FirstSeen: now.Add(-15 * 24 * time.Hour), LastSeen: now.Add(-3 * time.Hour)},
		{Raw: "go test ./...", Binary: "go", Subcommand: "test", Flags: "[]", Category: "package", Frequency: 17, FirstSeen: now.Add(-25 * 24 * time.Hour), LastSeen: now.Add(-4 * time.Hour)},
		{Raw: "find . -name '*.log' -mtime +30 -delete", Binary: "find", Subcommand: "", Flags: `["-name", "-mtime", "-delete"]`, Category: "filesystem", Frequency: 2, FirstSeen: now.Add(-60 * 24 * time.Hour), LastSeen: now.Add(-45 * 24 * time.Hour)},
		{Raw: "openssl req -x509 -newkey rsa:4096 -keyout key.pem -out cert.pem -days 365", Binary: "openssl", Subcommand: "req", Flags: `["-x509", "-newkey", "-keyout", "-out", "-days"]`, Category: "other", Frequency: 1, FirstSeen: now.Add(-90 * 24 * time.Hour), LastSeen: now.Add(-90 * 24 * time.Hour)},
		{Raw: "git status", Binary: "git", Subcommand: "status", Flags: "[]", Category: "git", Frequency: 100, FirstSeen: now.Add(-30 * 24 * time.Hour), LastSeen: now},
		{Raw: "git push", Binary: "git", Subcommand: "push", Flags: "[]", Category: "git", Frequency: 30, FirstSeen: now.Add(-30 * 24 * time.Hour), LastSeen: now.Add(-1 * time.Hour)},
		{Raw: "npm install", Binary: "npm", Subcommand: "install", Flags: "[]", Category: "package", Frequency: 8, FirstSeen: now.Add(-10 * 24 * time.Hour), LastSeen: now.Add(-5 * time.Hour)},
		{Raw: "ssh user@server.example.com", Binary: "ssh", Subcommand: "", Flags: "[]", Category: "network", Frequency: 5, FirstSeen: now.Add(-20 * 24 * time.Hour), LastSeen: now.Add(-7 * 24 * time.Hour)},
	}
	for _, cmd := range cmds {
		store.InsertCommand(&cmd)
	}
}

// --- Stats computation tests ---

func TestStatsGetStats(t *testing.T) {
	store, cleanup := setupTestVault(t)
	defer cleanup()
	seedVault(t, store)

	stats, err := store.GetStats()
	if err != nil {
		t.Fatalf("GetStats failed: %v", err)
	}

	if stats.UniqueCommands != 10 {
		t.Errorf("expected 10 unique commands, got %d", stats.UniqueCommands)
	}
	if stats.TotalCommands <= 0 {
		t.Error("expected positive total commands")
	}
}

func TestStatsGetTopCommands(t *testing.T) {
	store, cleanup := setupTestVault(t)
	defer cleanup()
	seedVault(t, store)

	top, err := store.GetTopCommands(30, 5)
	if err != nil {
		t.Fatalf("GetTopCommands failed: %v", err)
	}

	if len(top) == 0 {
		t.Fatal("expected at least 1 top command")
	}

	// Should be sorted by frequency descending
	for i := 1; i < len(top); i++ {
		if top[i].Frequency > top[i-1].Frequency {
			t.Errorf("top commands not sorted by frequency: %d > %d at index %d", top[i].Frequency, top[i-1].Frequency, i)
		}
	}
}

func TestStatsGetCategories(t *testing.T) {
	store, cleanup := setupTestVault(t)
	defer cleanup()
	seedVault(t, store)

	cats, err := store.GetCategories()
	if err != nil {
		t.Fatalf("GetCategories failed: %v", err)
	}

	if len(cats) < 3 {
		t.Fatalf("expected at least 3 categories, got %d", len(cats))
	}

	// git should be the top category
	if cats[0].Category != "git" {
		t.Errorf("expected git as top category, got %s", cats[0].Category)
	}
}

func TestStatsGetRareCommands(t *testing.T) {
	store, cleanup := setupTestVault(t)
	defer cleanup()
	seedVault(t, store)

	rare, err := store.GetRareCommands(3, 10)
	if err != nil {
		t.Fatalf("GetRareCommands failed: %v", err)
	}

	for _, cmd := range rare {
		if cmd.Frequency > 3 {
			t.Errorf("rare command has frequency %d > 3: %s", cmd.Frequency, cmd.Raw)
		}
	}
}

func TestStatsGetVaultPeriod(t *testing.T) {
	store, cleanup := setupTestVault(t)
	defer cleanup()
	seedVault(t, store)

	first, last, err := store.GetVaultPeriod()
	if err != nil {
		t.Fatalf("GetVaultPeriod failed: %v", err)
	}

	if first.IsZero() || last.IsZero() {
		t.Error("expected non-zero vault period times")
	}

	if !last.After(first) && !last.Equal(first) {
		t.Errorf("expected last (%v) >= first (%v)", last, first)
	}
}

// --- Alias suggestion tests ---

func TestGenerateAliasKnownCommands(t *testing.T) {
	tests := []struct {
		raw      string
		binary   string
		sub      string
		expected string
	}{
		{"git status", "git", "status", "gst"},
		{"git commit -m 'test'", "git", "commit", "gc"},
		{"docker compose up -d", "docker", "compose up", "dcud"},
		{"docker ps -a", "docker", "ps", "dpsa"},
		{"kubectl get pods", "kubectl", "get pods", "kgp"},
		{"npm install", "npm", "install", "ni"},
		{"go test ./...", "go", "test", "gt"},
		{"git log --oneline --graph --all", "git", "log", "glga"},
	}

	for _, tc := range tests {
		t.Run(tc.raw, func(t *testing.T) {
			cmds := []vault.Command{
				{Raw: tc.raw, Binary: tc.binary, Subcommand: tc.sub, Frequency: 50},
			}
			suggestions := generateTestAliases(cmds)
			if len(suggestions) == 0 {
				t.Fatalf("expected alias suggestion for '%s'", tc.raw)
			}
			if suggestions[0].Alias != tc.expected {
				t.Errorf("expected alias '%s', got '%s'", tc.expected, suggestions[0].Alias)
			}
		})
	}
}

func TestGenerateAliasNoSuggestionForShortCommands(t *testing.T) {
	cmds := []vault.Command{
		{Raw: "ls", Binary: "ls", Frequency: 100},
		{Raw: "pwd", Binary: "pwd", Frequency: 50},
	}
	suggestions := generateTestAliases(cmds)
	if len(suggestions) != 0 {
		t.Errorf("expected no suggestions for short commands, got %d", len(suggestions))
	}
}

func TestGenerateAliasNoDuplicates(t *testing.T) {
	cmds := []vault.Command{
		{Raw: "git status", Binary: "git", Subcommand: "status", Frequency: 50},
		{Raw: "git status", Binary: "git", Subcommand: "status", Frequency: 30},
	}
	suggestions := generateTestAliases(cmds)
	seen := make(map[string]int)
	for _, s := range suggestions {
		seen[s.Alias]++
		if seen[s.Alias] > 1 {
			t.Errorf("duplicate alias: %s", s.Alias)
		}
	}
}

func TestGenerateAliasSkipsRecallCommands(t *testing.T) {
	cmds := []vault.Command{
		{Raw: "recall search something", Binary: "recall", Subcommand: "search", Frequency: 50},
	}
	suggestions := generateTestAliases(cmds)
	if len(suggestions) != 0 {
		t.Errorf("expected no suggestions for recall commands, got %d", len(suggestions))
	}
}

// --- Vault browser model tests ---

func TestVaultBrowserModelInit(t *testing.T) {
	store, cleanup := setupTestVault(t)
	defer cleanup()
	seedVault(t, store)

	model := ui.NewVaultBrowser(store, "", "recency")
	if model.Init() == nil {
		t.Error("expected Init to return a command")
	}
}

func TestVaultBrowserSortModes(t *testing.T) {
	tests := []struct {
		sort     string
		expected string
	}{
		{"frequency", "frequency"},
		{"recency", "recency"},
		{"alpha", "alpha"},
		{"", "recency"},
	}

	store, cleanup := setupTestVault(t)
	defer cleanup()

	for _, tc := range tests {
		t.Run(tc.sort, func(t *testing.T) {
			_ = ui.NewVaultBrowser(store, "", tc.sort)
		})
	}
}

func TestGetAllCommandsSorting(t *testing.T) {
	store, cleanup := setupTestVault(t)
	defer cleanup()
	seedVault(t, store)

	// Frequency sort
	cmds, err := store.GetAllCommands("frequency", 100)
	if err != nil {
		t.Fatalf("GetAllCommands(frequency) failed: %v", err)
	}
	if len(cmds) == 0 {
		t.Fatal("expected commands")
	}
	for i := 1; i < len(cmds); i++ {
		if cmds[i].Frequency > cmds[i-1].Frequency {
			t.Errorf("frequency sort violated at index %d: %d > %d", i, cmds[i].Frequency, cmds[i-1].Frequency)
		}
	}

	// Alpha sort
	cmds, err = store.GetAllCommands("alpha", 100)
	if err != nil {
		t.Fatalf("GetAllCommands(alpha) failed: %v", err)
	}
	for i := 1; i < len(cmds); i++ {
		if cmds[i].Raw < cmds[i-1].Raw {
			t.Errorf("alpha sort violated at index %d: %s < %s", i, cmds[i].Raw, cmds[i-1].Raw)
		}
	}
}

func TestGetCommandsByCategory(t *testing.T) {
	store, cleanup := setupTestVault(t)
	defer cleanup()
	seedVault(t, store)

	cmds, err := store.GetCommandsByCategory("git", 100)
	if err != nil {
		t.Fatalf("GetCommandsByCategory failed: %v", err)
	}

	for _, cmd := range cmds {
		if cmd.Category != "git" {
			t.Errorf("expected category 'git', got '%s'", cmd.Category)
		}
	}

	if len(cmds) < 3 {
		t.Errorf("expected at least 3 git commands, got %d", len(cmds))
	}
}

func TestDeleteCommand(t *testing.T) {
	store, cleanup := setupTestVault(t)
	defer cleanup()
	seedVault(t, store)

	cmds, _ := store.GetAllCommands("recency", 1)
	if len(cmds) == 0 {
		t.Fatal("expected at least one command")
	}

	statsBefore, _ := store.GetStats()

	err := store.DeleteCommand(cmds[0].ID)
	if err != nil {
		t.Fatalf("DeleteCommand failed: %v", err)
	}

	statsAfter, _ := store.GetStats()
	if statsAfter.UniqueCommands >= statsBefore.UniqueCommands {
		t.Error("expected unique count to decrease after delete")
	}
}

func TestGetHighFrequencyCommands(t *testing.T) {
	store, cleanup := setupTestVault(t)
	defer cleanup()
	seedVault(t, store)

	cmds, err := store.GetHighFrequencyCommands(20)
	if err != nil {
		t.Fatalf("GetHighFrequencyCommands failed: %v", err)
	}

	for _, cmd := range cmds {
		if cmd.Frequency < 20 {
			t.Errorf("expected frequency >= 20, got %d for '%s'", cmd.Frequency, cmd.Raw)
		}
	}
}

// --- Helper to reuse suggest logic in tests ---

type testAliasSuggestion struct {
	Command   string
	Alias     string
	Frequency int
}

func generateTestAliases(cmds []vault.Command) []testAliasSuggestion {
	knownAliases := map[string]string{
		"git status":                       "gst",
		"git add":                          "ga",
		"git add -A":                       "gaa",
		"git commit":                       "gc",
		"git push":                         "gp",
		"git pull":                         "gpl",
		"git pull --rebase":                "gpr",
		"git checkout":                     "gco",
		"git branch":                       "gb",
		"git log --oneline":                "glog",
		"git log --oneline --graph":        "glg",
		"git log --oneline --graph --all":  "glga",
		"git diff":                         "gd",
		"git diff --cached":                "gdc",
		"git stash":                        "gsta",
		"git stash pop":                    "gstp",
		"git merge":                        "gm",
		"git rebase":                       "grb",
		"docker ps":                        "dps",
		"docker ps -a":                     "dpsa",
		"docker images":                    "dim",
		"docker compose up":               "dcu",
		"docker compose up -d":            "dcud",
		"docker compose down":             "dcd",
		"docker compose logs":             "dcl",
		"docker system prune":             "dsp",
		"kubectl get pods":                "kgp",
		"kubectl get services":            "kgs",
		"kubectl get deployments":         "kgd",
		"kubectl get namespaces":          "kgns",
		"kubectl describe":                "kd",
		"kubectl logs":                    "kl",
		"kubectl apply -f":                "kaf",
		"npm install":                     "ni",
		"npm run":                         "nr",
		"npm test":                        "nt",
		"go test ./...":                   "gt",
		"go build":                        "gob",
		"go run":                          "gor",
		"python3":                         "py",
		"python":                          "py",
	}

	var suggestions []testAliasSuggestion
	seen := make(map[string]bool)

	for _, cmd := range cmds {
		raw := cmd.Raw
		if len(raw) <= 5 {
			continue
		}
		if len(raw) >= 7 && raw[:7] == "recall " {
			continue
		}

		// Match longest prefix first (exact match > prefix+space match)
		alias := ""
		bestLen := 0
		for prefix, a := range knownAliases {
			if raw == prefix {
				if len(prefix) > bestLen {
					bestLen = len(prefix)
					alias = a
				}
			} else if len(raw) > len(prefix) && raw[:len(prefix)] == prefix && raw[len(prefix)] == ' ' {
				if len(prefix) > bestLen {
					bestLen = len(prefix)
					alias = a
				}
			}
		}

		if alias == "" {
			continue
		}
		if alias == cmd.Binary || len(alias) >= len(raw) {
			continue
		}

		if !seen[alias] {
			seen[alias] = true
			suggestions = append(suggestions, testAliasSuggestion{
				Command:   raw,
				Alias:     alias,
				Frequency: cmd.Frequency,
			})
		}
	}

	return suggestions
}

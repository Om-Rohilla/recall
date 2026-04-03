package tests

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/Om-Rohilla/recall/internal/vault"
)

func tempVault(t *testing.T) *vault.Store {
	t.Helper()
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "test_vault.db")
	store, err := vault.NewStore(dbPath)
	if err != nil {
		t.Fatalf("failed to create test vault: %v", err)
	}
	t.Cleanup(func() { store.Close() })
	return store
}

func TestSchemaCreation(t *testing.T) {
	store := tempVault(t)

	stats, err := store.GetStats()
	if err != nil {
		t.Fatalf("GetStats failed: %v", err)
	}
	if stats.TotalCommands != 0 || stats.UniqueCommands != 0 {
		t.Errorf("expected empty vault, got total=%d unique=%d", stats.TotalCommands, stats.UniqueCommands)
	}
}

func TestInsertCommand(t *testing.T) {
	store := tempVault(t)

	now := time.Now().UTC()
	exitCode := 0
	cmd := &vault.Command{
		Raw:        "git status",
		Binary:     "git",
		Subcommand: "status",
		Flags:      "[]",
		Category:   "git",
		FirstSeen:  now,
		LastSeen:   now,
		LastExit:   &exitCode,
	}

	id, err := store.InsertCommand(cmd)
	if err != nil {
		t.Fatalf("InsertCommand failed: %v", err)
	}
	if id <= 0 {
		t.Errorf("expected positive ID, got %d", id)
	}

	fetched, err := store.GetCommand(id)
	if err != nil {
		t.Fatalf("GetCommand failed: %v", err)
	}
	if fetched.Raw != "git status" {
		t.Errorf("expected raw='git status', got '%s'", fetched.Raw)
	}
	if fetched.Binary != "git" {
		t.Errorf("expected binary='git', got '%s'", fetched.Binary)
	}
	if fetched.Frequency != 1 {
		t.Errorf("expected frequency=1, got %d", fetched.Frequency)
	}
}

func TestInsertDuplicateUpdatesFrequency(t *testing.T) {
	store := tempVault(t)

	now := time.Now().UTC()
	exitCode := 0
	cmd := &vault.Command{
		Raw:       "docker ps",
		Binary:    "docker",
		Flags:     "[]",
		Category:  "docker",
		FirstSeen: now,
		LastSeen:  now,
		LastExit:  &exitCode,
	}

	id1, err := store.InsertCommand(cmd)
	if err != nil {
		t.Fatalf("first insert failed: %v", err)
	}

	id2, err := store.InsertCommand(cmd)
	if err != nil {
		t.Fatalf("second insert failed: %v", err)
	}

	if id1 != id2 {
		t.Errorf("expected same ID for duplicate, got %d and %d", id1, id2)
	}

	fetched, err := store.GetCommand(id1)
	if err != nil {
		t.Fatalf("GetCommand failed: %v", err)
	}
	if fetched.Frequency != 2 {
		t.Errorf("expected frequency=2 after duplicate, got %d", fetched.Frequency)
	}
}

func TestFTS5Search(t *testing.T) {
	store := tempVault(t)

	now := time.Now().UTC()
	commands := []vault.Command{
		{Raw: "find . -type f -size +100M", Binary: "find", Category: "filesystem", FirstSeen: now, LastSeen: now, Flags: "[]"},
		{Raw: "docker compose up -d", Binary: "docker", Subcommand: "compose up", Category: "docker", FirstSeen: now, LastSeen: now, Flags: "[]"},
		{Raw: "git log --oneline", Binary: "git", Subcommand: "log", Category: "git", FirstSeen: now, LastSeen: now, Flags: "[]"},
		{Raw: "find . -name '*.log' -delete", Binary: "find", Category: "filesystem", FirstSeen: now, LastSeen: now, Flags: "[]"},
		{Raw: "kubectl get pods -n staging", Binary: "kubectl", Subcommand: "get pods", Category: "kubernetes", FirstSeen: now, LastSeen: now, Flags: "[]"},
	}

	inserted, err := store.BatchInsertCommands(commands)
	if err != nil {
		t.Fatalf("BatchInsertCommands failed: %v", err)
	}
	if inserted != 5 {
		t.Errorf("expected 5 inserted, got %d", inserted)
	}

	if err := store.RebuildFTSIndex(); err != nil {
		t.Fatalf("RebuildFTSIndex failed: %v", err)
	}

	tests := []struct {
		query    string
		wantMin  int
		wantCmd  string
	}{
		{"find", 2, "find"},
		{"docker", 1, "docker"},
		{"git", 1, "git"},
		{"pods", 1, "kubectl"},
		{"filesystem", 2, "find"},
	}

	for _, tt := range tests {
		t.Run(tt.query, func(t *testing.T) {
			results, err := store.SearchFTS5(tt.query, 10)
			if err != nil {
				t.Fatalf("SearchFTS5(%q) failed: %v", tt.query, err)
			}
			if len(results) < tt.wantMin {
				t.Errorf("SearchFTS5(%q): expected at least %d results, got %d", tt.query, tt.wantMin, len(results))
			}
			if len(results) > 0 && results[0].Command.Binary != tt.wantCmd {
				t.Errorf("SearchFTS5(%q): expected top result binary=%q, got %q", tt.query, tt.wantCmd, results[0].Command.Binary)
			}
		})
	}
}

func TestInsertContext(t *testing.T) {
	store := tempVault(t)

	now := time.Now().UTC()
	cmd := &vault.Command{
		Raw:       "make build",
		Binary:    "make",
		Flags:     "[]",
		Category:  "build",
		FirstSeen: now,
		LastSeen:  now,
	}

	id, err := store.InsertCommand(cmd)
	if err != nil {
		t.Fatalf("InsertCommand failed: %v", err)
	}

	exitCode := 0
	ctx := &vault.Context{
		CommandID:   id,
		Cwd:         "/home/user/project",
		GitRepo:     "myproject",
		GitBranch:   "main",
		ProjectType: "go",
		Timestamp:   now,
		ExitCode:    &exitCode,
		SessionID:   "test-session",
	}

	if err := store.InsertContext(ctx); err != nil {
		t.Fatalf("InsertContext failed: %v", err)
	}

	stats, err := store.GetStats()
	if err != nil {
		t.Fatalf("GetStats failed: %v", err)
	}
	if stats.TotalContexts != 1 {
		t.Errorf("expected 1 context, got %d", stats.TotalContexts)
	}
}

func TestGetRecentCommands(t *testing.T) {
	store := tempVault(t)

	now := time.Now().UTC()
	for i := 0; i < 5; i++ {
		cmd := &vault.Command{
			Raw:       "command-" + string(rune('a'+i)),
			Binary:    "cmd",
			Flags:     "[]",
			FirstSeen: now.Add(time.Duration(i) * time.Minute),
			LastSeen:  now.Add(time.Duration(i) * time.Minute),
		}
		if _, err := store.InsertCommand(cmd); err != nil {
			t.Fatalf("InsertCommand failed: %v", err)
		}
	}

	recent, err := store.GetRecentCommands(3)
	if err != nil {
		t.Fatalf("GetRecentCommands failed: %v", err)
	}
	if len(recent) != 3 {
		t.Errorf("expected 3 recent commands, got %d", len(recent))
	}
}

func TestBatchInsert(t *testing.T) {
	store := tempVault(t)

	now := time.Now().UTC()
	cmds := []vault.Command{
		{Raw: "ls -la", Binary: "ls", Category: "filesystem", Flags: "[]", FirstSeen: now, LastSeen: now},
		{Raw: "git status", Binary: "git", Category: "git", Flags: "[]", FirstSeen: now, LastSeen: now},
		{Raw: "docker ps", Binary: "docker", Category: "docker", Flags: "[]", FirstSeen: now, LastSeen: now},
	}

	inserted, err := store.BatchInsertCommands(cmds)
	if err != nil {
		t.Fatalf("BatchInsertCommands failed: %v", err)
	}
	if inserted != 3 {
		t.Errorf("expected 3 inserted, got %d", inserted)
	}

	stats, err := store.GetStats()
	if err != nil {
		t.Fatalf("GetStats failed: %v", err)
	}
	if stats.UniqueCommands != 3 {
		t.Errorf("expected 3 unique commands, got %d", stats.UniqueCommands)
	}
}

func TestFTSIndexIntegrity(t *testing.T) {
	store := tempVault(t)

	now := time.Now().UTC()
	cmd := &vault.Command{
		Raw:       "echo hello",
		Binary:    "echo",
		Flags:     "[]",
		FirstSeen: now,
		LastSeen:  now,
	}
	if _, err := store.InsertCommand(cmd); err != nil {
		t.Fatalf("InsertCommand failed: %v", err)
	}

	if err := store.IntegrityCheck(); err != nil {
		t.Fatalf("IntegrityCheck failed: %v", err)
	}
}

func TestEmptyVaultPath(t *testing.T) {
	dir := t.TempDir()
	nestedPath := filepath.Join(dir, "deep", "nested", "path", "vault.db")

	store, err := vault.NewStore(nestedPath)
	if err != nil {
		t.Fatalf("NewStore with nested path failed: %v", err)
	}
	if err := store.Close(); err != nil {
		t.Fatalf("closing store failed: %v", err)
	}

	if _, err := os.Stat(nestedPath + ".enc"); os.IsNotExist(err) {
		t.Error("expected vault.db.enc to exist at nested path after closing with default encryption")
	}
}

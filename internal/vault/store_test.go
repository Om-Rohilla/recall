package vault

import (
	"os"
	"testing"
	"time"
)

func tempStore(t *testing.T) *Store {
	t.Helper()
	tmpFile, err := os.CreateTemp(t.TempDir(), "vault-test-*.db")
	if err != nil {
		t.Fatalf("creating temp file: %v", err)
	}
	tmpFile.Close()

	store, err := NewStore(tmpFile.Name())
	if err != nil {
		t.Fatalf("creating store: %v", err)
	}
	t.Cleanup(func() { store.Close() })
	return store
}

func TestNewStore_CreatesDB(t *testing.T) {
	store := tempStore(t)
	if store == nil {
		t.Fatal("store should not be nil")
	}
}

func TestInsertCommand_Basic(t *testing.T) {
	store := tempStore(t)

	cmd := &Command{
		Raw:       "git status",
		Binary:    "git",
		Category:  "git",
		Frequency: 1,
		FirstSeen: time.Now().UTC(),
		LastSeen:  time.Now().UTC(),
	}

	id, err := store.InsertCommand(cmd)
	if err != nil {
		t.Fatalf("inserting command: %v", err)
	}
	if id <= 0 {
		t.Errorf("expected positive ID, got %d", id)
	}
}

func TestInsertCommand_DuplicateIncrementsFrequency(t *testing.T) {
	store := tempStore(t)
	now := time.Now().UTC()

	cmd := &Command{
		Raw:       "git status",
		Binary:    "git",
		Category:  "git",
		Frequency: 1,
		FirstSeen: now,
		LastSeen:  now,
	}

	_, err := store.InsertCommand(cmd)
	if err != nil {
		t.Fatalf("first insert: %v", err)
	}

	_, err = store.InsertCommand(cmd)
	if err != nil {
		t.Fatalf("second insert: %v", err)
	}

	// Fetch and verify frequency
	cmds, err := store.GetAllCommands("recency", 10)
	if err != nil {
		t.Fatalf("getting commands: %v", err)
	}
	if len(cmds) != 1 {
		t.Fatalf("expected 1 command (deduplicated), got %d", len(cmds))
	}
	if cmds[0].Frequency < 2 {
		t.Errorf("expected frequency >= 2 after duplicate insert, got %d", cmds[0].Frequency)
	}
}

func TestInsertContext(t *testing.T) {
	store := tempStore(t)
	now := time.Now().UTC()

	cmd := &Command{
		Raw:       "docker ps",
		Binary:    "docker",
		Category:  "docker",
		Frequency: 1,
		FirstSeen: now,
		LastSeen:  now,
	}
	cmdID, _ := store.InsertCommand(cmd)

	exitCode := 0
	ctx := &Context{
		CommandID: cmdID,
		Cwd:       "/home/user/project",
		GitRepo:   "myrepo",
		GitBranch: "main",
		Timestamp: now,
		ExitCode:  &exitCode,
	}

	err := store.InsertContext(ctx)
	if err != nil {
		t.Fatalf("inserting context: %v", err)
	}

	ctxs, err := store.GetContextsForCommand(cmdID)
	if err != nil {
		t.Fatalf("getting contexts: %v", err)
	}
	if len(ctxs) != 1 {
		t.Fatalf("expected 1 context, got %d", len(ctxs))
	}
	if ctxs[0].GitRepo != "myrepo" {
		t.Errorf("expected git repo 'myrepo', got %q", ctxs[0].GitRepo)
	}
}

func TestSearchFTS5_Basic(t *testing.T) {
	store := tempStore(t)
	now := time.Now().UTC()

	commands := []string{
		"git status",
		"git commit -m 'fix bug'",
		"docker ps -a",
		"kubectl get pods",
		"find / -name '*.log' -size +100M",
	}

	for _, raw := range commands {
		cmd := &Command{
			Raw:       raw,
			Binary:    "test",
			Category:  "test",
			Frequency: 1,
			FirstSeen: now,
			LastSeen:  now,
		}
		if _, err := store.InsertCommand(cmd); err != nil {
			t.Fatalf("inserting %q: %v", raw, err)
		}
	}

	results, err := store.SearchFTS5("git", 10)
	if err != nil {
		t.Fatalf("searching: %v", err)
	}

	if !hasFTSTable(store.db, "commands_fts") {
		t.Skip("FTS5 missing, skipping remaining FTS test logic")
	}

	if len(results) < 2 {
		t.Errorf("expected at least 2 git results, got %d", len(results))
	}
	for _, r := range results {
		if r.Confidence <= 0 {
			t.Errorf("confidence should be positive, got %f", r.Confidence)
		}
	}
}

func TestSearchFTS5_NoResults(t *testing.T) {
	store := tempStore(t)

	results, err := store.SearchFTS5("nonexistentcommandxyz", 10)
	if err != nil {
		t.Fatalf("searching: %v", err)
	}
	if len(results) != 0 {
		t.Errorf("expected 0 results, got %d", len(results))
	}
}

func TestGetStats(t *testing.T) {
	store := tempStore(t)
	now := time.Now().UTC()

	for i := 0; i < 5; i++ {
		cmd := &Command{
			Raw:       "cmd" + string(rune('A'+i)),
			Binary:    "test",
			Category:  "test",
			Frequency: i + 1,
			FirstSeen: now,
			LastSeen:  now,
		}
		store.InsertCommand(cmd)
	}

	stats, err := store.GetStats()
	if err != nil {
		t.Fatalf("getting stats: %v", err)
	}
	if stats.UniqueCommands != 5 {
		t.Errorf("expected 5 unique commands, got %d", stats.UniqueCommands)
	}
}

func TestGetCategories(t *testing.T) {
	store := tempStore(t)
	now := time.Now().UTC()

	cats := []string{"git", "git", "docker", "network"}
	for i, cat := range cats {
		cmd := &Command{
			Raw:       "cmd" + string(rune('A'+i)),
			Binary:    "test",
			Category:  cat,
			Frequency: 1,
			FirstSeen: now,
			LastSeen:  now,
		}
		store.InsertCommand(cmd)
	}

	categories, err := store.GetCategories()
	if err != nil {
		t.Fatalf("getting categories: %v", err)
	}
	if len(categories) < 2 {
		t.Errorf("expected at least 2 categories, got %d", len(categories))
	}
}

func TestDeleteCommand(t *testing.T) {
	store := tempStore(t)
	now := time.Now().UTC()

	cmd := &Command{
		Raw:       "git status",
		Binary:    "git",
		Category:  "git",
		Frequency: 1,
		FirstSeen: now,
		LastSeen:  now,
	}
	id, _ := store.InsertCommand(cmd)

	err := store.DeleteCommand(id)
	if err != nil {
		t.Fatalf("deleting command: %v", err)
	}

	cmds, _ := store.GetAllCommands("recency", 10)
	if len(cmds) != 0 {
		t.Errorf("expected 0 commands after delete, got %d", len(cmds))
	}
}

func TestSanitizeFTSQuery(t *testing.T) {
	tests := []struct {
		input    string
		wantSafe bool
	}{
		{"normal query", true},
		{"git status", true},
		{"find -name '*.go'", true},
		{"", true},
		{"OR AND NOT", true}, // operators should be sanitized
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := sanitizeFTSQuery(tt.input)
			_ = result // just verify no panic
		})
	}
}

func TestGetCurrentStreak(t *testing.T) {
	store := tempStore(t)
	now := time.Now().UTC()

	// Insert commands for today and yesterday
	for i := 0; i < 2; i++ {
		cmd := &Command{
			Raw:       "git status",
			Binary:    "git",
			Category:  "git",
			Frequency: 1,
			FirstSeen: now.Add(-time.Duration(i) * 24 * time.Hour),
			LastSeen:  now.Add(-time.Duration(i) * 24 * time.Hour),
		}
		store.InsertCommand(cmd)
	}

	streak, err := store.GetCurrentStreak()
	if err != nil {
		t.Fatalf("getting streak: %v", err)
	}
	if streak.CurrentStreak < 1 {
		t.Errorf("expected streak >= 1, got %d", streak.CurrentStreak)
	}
	if streak.StreakEmoji == "" {
		t.Error("streak emoji should not be empty")
	}
}

func TestRebuildFTSIndex(t *testing.T) {
	store := tempStore(t)
	now := time.Now().UTC()

	cmd := &Command{
		Raw:       "test rebuild",
		Binary:    "test",
		Category:  "test",
		Frequency: 1,
		FirstSeen: now,
		LastSeen:  now,
	}
	store.InsertCommand(cmd)

	err := store.RebuildFTSIndex()
	if err != nil {
		t.Fatalf("rebuilding FTS index: %v", err)
	}

	// Verify search still works after rebuild
	results, err := store.SearchFTS5("test", 10)
	if err != nil {
		t.Fatalf("searching after rebuild: %v", err)
	}

	if !hasFTSTable(store.db, "commands_fts") {
		t.Skip("FTS5 missing, skipping assertion")
	}

	if len(results) == 0 {
		t.Error("expected results after FTS rebuild")
	}
}

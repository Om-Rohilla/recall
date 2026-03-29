package tests

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	appctx "github.com/Om-Rohilla/recall/internal/context"
	"github.com/Om-Rohilla/recall/internal/intelligence"
	"github.com/Om-Rohilla/recall/internal/vault"
)

func TestIntentExtraction(t *testing.T) {
	tests := []struct {
		query       string
		wantTokens  int
		wantAction  string
		wantTarget  string
		wantCategory string
	}{
		{"find large files", 2, "find", "file", "filesystem"},
		{"kill process on port 3000", 2, "stop", "process", "process"},
		{"undo last git commit", 2, "undo", "commit", "git"},
		{"docker cleanup containers", 2, "delete", "container", "docker"},
		{"list kubernetes pods", 2, "show", "pod", "kubernetes"},
		{"compress folder to tar", 2, "archive", "directory", "archive"},
		{"show disk usage", 2, "show", "disk", "filesystem"},
		{"ssh connect to server", 2, "connect", "", "network"},
		{"create new branch", 2, "create", "branch", "git"},
		{"install npm package", 2, "install", "package", "package"},
	}

	for _, tt := range tests {
		t.Run(tt.query, func(t *testing.T) {
			intent := intelligence.ExtractIntent(tt.query)

			if len(intent.Tokens) < tt.wantTokens {
				t.Errorf("ExtractIntent(%q).Tokens = %v (len %d), want at least %d tokens",
					tt.query, intent.Tokens, len(intent.Tokens), tt.wantTokens)
			}
			if intent.Action != tt.wantAction {
				t.Errorf("ExtractIntent(%q).Action = %q, want %q", tt.query, intent.Action, tt.wantAction)
			}
			if tt.wantTarget != "" && intent.Target != tt.wantTarget {
				t.Errorf("ExtractIntent(%q).Target = %q, want %q", tt.query, intent.Target, tt.wantTarget)
			}
			if tt.wantCategory != "" && intent.Category != tt.wantCategory {
				t.Errorf("ExtractIntent(%q).Category = %q, want %q", tt.query, intent.Category, tt.wantCategory)
			}
		})
	}
}

func TestSynonymExpansion(t *testing.T) {
	intent := intelligence.ExtractIntent("find big files")
	expanded := intent.ExpandedTokens

	wantContains := []string{"find", "big", "large", "search", "locate"}
	for _, want := range wantContains {
		found := false
		for _, got := range expanded {
			if got == want {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expanded tokens %v should contain %q", expanded, want)
		}
	}
}

func TestStopWordRemoval(t *testing.T) {
	intent := intelligence.ExtractIntent("how do I find the large files in my directory")
	for _, token := range intent.Tokens {
		stopWords := []string{"how", "do", "i", "the", "in", "my"}
		for _, sw := range stopWords {
			if token == sw {
				t.Errorf("token %q should have been filtered as a stop word", sw)
			}
		}
	}
}

func TestBuildFTSQuery(t *testing.T) {
	intent := intelligence.ExtractIntent("find large files")
	fts := intelligence.BuildFTSQuery(intent)
	if fts == "" {
		t.Fatal("BuildFTSQuery returned empty string for valid intent")
	}
	if len(fts) < 10 {
		t.Errorf("FTS query too short: %q", fts)
	}
}

func TestEmptyQuery(t *testing.T) {
	intent := intelligence.ExtractIntent("")
	if len(intent.Tokens) != 0 {
		t.Errorf("empty query should produce no tokens, got %v", intent.Tokens)
	}
	fts := intelligence.BuildFTSQuery(intent)
	if fts != "" {
		t.Errorf("empty intent should produce empty FTS query, got %q", fts)
	}
}

func TestScorerWeightsSum(t *testing.T) {
	total := intelligence.WeightText + intelligence.WeightIntent +
		intelligence.WeightFrequency + intelligence.WeightContext + intelligence.WeightRecency
	if total < 0.99 || total > 1.01 {
		t.Errorf("scoring weights should sum to 1.0, got %f", total)
	}
}

func TestScorerBasicRanking(t *testing.T) {
	intent := intelligence.ExtractIntent("find large files")
	ctx := appctx.CurrentContext{
		Cwd: "/home/user/project",
		Project: appctx.ProjectInfo{Type: "go"},
	}

	now := time.Now().UTC()

	highRelevance := intelligence.ScoringInput{
		Command: vault.Command{
			Raw:       "find . -type f -size +100M",
			Binary:    "find",
			Category:  "filesystem",
			Frequency: 10,
			LastSeen:  now,
		},
		Contexts: []vault.Context{
			{Cwd: "/home/user/project", ProjectType: "go"},
		},
		FTSRank:      5.0,
		MaxFrequency: 20,
		MatchType:    "vault",
	}

	lowRelevance := intelligence.ScoringInput{
		Command: vault.Command{
			Raw:       "echo hello world",
			Binary:    "echo",
			Category:  "other",
			Frequency: 1,
			LastSeen:  now.Add(-90 * 24 * time.Hour),
		},
		FTSRank:      0.5,
		MaxFrequency: 20,
		MatchType:    "vault",
	}

	highResult := intelligence.ScoreCandidate(highRelevance, intent, ctx)
	lowResult := intelligence.ScoreCandidate(lowRelevance, intent, ctx)

	if highResult.FinalScore <= lowResult.FinalScore {
		t.Errorf("'find' command (%.3f) should score higher than 'echo' (%.3f) for query 'find large files'",
			highResult.FinalScore, lowResult.FinalScore)
	}

	if highResult.Confidence <= 0 || highResult.Confidence > 100 {
		t.Errorf("confidence should be 0-100, got %f", highResult.Confidence)
	}
}

func TestFrequencyBoost(t *testing.T) {
	intent := intelligence.ExtractIntent("docker ps")
	ctx := appctx.CurrentContext{}
	now := time.Now().UTC()

	highFreq := intelligence.ScoringInput{
		Command: vault.Command{
			Raw: "docker ps", Binary: "docker", Category: "docker",
			Frequency: 100, LastSeen: now,
		},
		FTSRank: 3.0, MaxFrequency: 100, MatchType: "vault",
	}

	lowFreq := intelligence.ScoringInput{
		Command: vault.Command{
			Raw: "docker ps -a", Binary: "docker", Category: "docker",
			Frequency: 1, LastSeen: now,
		},
		FTSRank: 3.0, MaxFrequency: 100, MatchType: "vault",
	}

	highResult := intelligence.ScoreCandidate(highFreq, intent, ctx)
	lowResult := intelligence.ScoreCandidate(lowFreq, intent, ctx)

	if highResult.FrequencyScore <= lowResult.FrequencyScore {
		t.Errorf("high frequency (%.3f) should score higher than low (%.3f)",
			highResult.FrequencyScore, lowResult.FrequencyScore)
	}
}

func TestRecencyBoost(t *testing.T) {
	intent := intelligence.ExtractIntent("git status")
	ctx := appctx.CurrentContext{}

	recent := intelligence.ScoringInput{
		Command: vault.Command{
			Raw: "git status", Binary: "git", Category: "git",
			Frequency: 5, LastSeen: time.Now().UTC(),
		},
		FTSRank: 3.0, MaxFrequency: 10, MatchType: "vault",
	}

	old := intelligence.ScoringInput{
		Command: vault.Command{
			Raw: "git status --short", Binary: "git", Category: "git",
			Frequency: 5, LastSeen: time.Now().UTC().Add(-180 * 24 * time.Hour),
		},
		FTSRank: 3.0, MaxFrequency: 10, MatchType: "vault",
	}

	recentResult := intelligence.ScoreCandidate(recent, intent, ctx)
	oldResult := intelligence.ScoreCandidate(old, intent, ctx)

	if recentResult.RecencyScore <= oldResult.RecencyScore {
		t.Errorf("recent command (%.3f) should score higher recency than old (%.3f)",
			recentResult.RecencyScore, oldResult.RecencyScore)
	}
}

func TestContextBoost(t *testing.T) {
	intent := intelligence.ExtractIntent("test")
	ctx := appctx.CurrentContext{
		Cwd:     "/home/user/myapp",
		Git:     appctx.GitState{IsRepo: true, RepoName: "myapp"},
		Project: appctx.ProjectInfo{Type: "go"},
	}

	matchingCtx := intelligence.ScoringInput{
		Command: vault.Command{
			Raw: "go test ./...", Binary: "go", Category: "package",
			Frequency: 5, LastSeen: time.Now().UTC(),
		},
		Contexts: []vault.Context{
			{Cwd: "/home/user/myapp", GitRepo: "myapp", ProjectType: "go"},
		},
		FTSRank: 2.0, MaxFrequency: 10, MatchType: "vault",
	}

	noCtx := intelligence.ScoringInput{
		Command: vault.Command{
			Raw: "pytest", Binary: "pytest", Category: "package",
			Frequency: 5, LastSeen: time.Now().UTC(),
		},
		FTSRank: 2.0, MaxFrequency: 10, MatchType: "vault",
	}

	matchResult := intelligence.ScoreCandidate(matchingCtx, intent, ctx)
	noCtxResult := intelligence.ScoreCandidate(noCtx, intent, ctx)

	if matchResult.ContextScore <= noCtxResult.ContextScore {
		t.Errorf("matching context (%.3f) should score higher than no context (%.3f)",
			matchResult.ContextScore, noCtxResult.ContextScore)
	}
}

func TestPatternExtraction(t *testing.T) {
	tests := []struct {
		input    string
		wantDiff bool
	}{
		{"find /home/user/docs -name '*.txt'", true},
		{"git checkout feature/auth-system", true},
		{"curl https://api.example.com/v1/users", true},
		{"ls -la", false},
		{"docker ps", false},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			pattern := intelligence.ExtractPattern(tt.input)
			if tt.wantDiff && pattern == tt.input {
				t.Errorf("ExtractPattern(%q) = same as input, expected templated version", tt.input)
			}
			if !tt.wantDiff && pattern != tt.input {
				t.Errorf("ExtractPattern(%q) = %q, expected unchanged", tt.input, pattern)
			}
		})
	}
}

func TestNormalizePattern(t *testing.T) {
	input := "  find  .  -type  f  "
	want := "find . -type f"
	got := intelligence.NormalizePattern(input)
	if got != want {
		t.Errorf("NormalizePattern(%q) = %q, want %q", input, got, want)
	}
}

func TestSearchEngineIntegration(t *testing.T) {
	store := tempVault(t)

	now := time.Now().UTC()
	commands := []vault.Command{
		{Raw: "find . -type f -size +100M", Binary: "find", Category: "filesystem", Frequency: 5, FirstSeen: now, LastSeen: now, Flags: "[]"},
		{Raw: "find . -name '*.log' -delete", Binary: "find", Category: "filesystem", Frequency: 2, FirstSeen: now, LastSeen: now, Flags: "[]"},
		{Raw: "docker compose up -d", Binary: "docker", Subcommand: "compose up", Category: "docker", Frequency: 20, FirstSeen: now, LastSeen: now, Flags: "[]"},
		{Raw: "git log --oneline", Binary: "git", Subcommand: "log", Category: "git", Frequency: 15, FirstSeen: now, LastSeen: now, Flags: "[]"},
		{Raw: "kubectl get pods -n staging", Binary: "kubectl", Subcommand: "get pods", Category: "kubernetes", Frequency: 8, FirstSeen: now, LastSeen: now, Flags: "[]"},
		{Raw: "lsof -ti:3000 | xargs kill", Binary: "lsof", Category: "process", Frequency: 3, FirstSeen: now, LastSeen: now, Flags: "[]"},
		{Raw: "tar -czf archive.tar.gz directory/", Binary: "tar", Category: "archive", Frequency: 4, FirstSeen: now, LastSeen: now, Flags: "[]"},
		{Raw: "ssh user@server.com", Binary: "ssh", Category: "network", Frequency: 10, FirstSeen: now, LastSeen: now, Flags: "[]"},
	}

	inserted, err := store.BatchInsertCommands(commands)
	if err != nil {
		t.Fatalf("BatchInsertCommands failed: %v", err)
	}
	if inserted != len(commands) {
		t.Fatalf("expected %d inserted, got %d", len(commands), inserted)
	}
	if err := store.RebuildFTSIndex(); err != nil {
		t.Fatalf("RebuildFTSIndex failed: %v", err)
	}

	ctx := appctx.CurrentContext{
		Cwd:     "/home/user/project",
		Project: appctx.ProjectInfo{Type: "go"},
	}

	tests := []struct {
		query   string
		wantCmd string
		wantMin int
	}{
		{"find large files", "find", 1},
		{"docker start services", "docker", 1},
		{"git history", "git", 1},
		{"kubernetes pods", "kubectl", 1},
		{"compress folder", "tar", 1},
		{"kill process on port", "lsof", 1},
	}

	for _, tt := range tests {
		t.Run(tt.query, func(t *testing.T) {
			opts := intelligence.SearchOptions{
				Limit:     5,
				VaultOnly: true,
			}
			results, err := intelligence.Search(store, tt.query, ctx, opts)
			if err != nil {
				t.Fatalf("Search(%q) failed: %v", tt.query, err)
			}
			if len(results) < tt.wantMin {
				t.Errorf("Search(%q): expected at least %d results, got %d", tt.query, tt.wantMin, len(results))
				return
			}
			if results[0].Command.Binary != tt.wantCmd {
				t.Errorf("Search(%q): top result binary = %q, want %q (raw: %q, confidence: %.1f%%)",
					tt.query, results[0].Command.Binary, tt.wantCmd, results[0].Command.Raw, results[0].Confidence)
			}
		})
	}
}

func TestSearchWithKnowledgeBase(t *testing.T) {
	store := tempVault(t)

	kbPath := findTestKBPath(t)
	if kbPath == "" {
		t.Skip("knowledge base not found, skipping")
	}

	inserted, err := intelligence.LoadKnowledgeBase(store, kbPath)
	if err != nil {
		t.Fatalf("LoadKnowledgeBase failed: %v", err)
	}
	if inserted == 0 {
		t.Fatal("expected knowledge base entries to be inserted")
	}

	ctx := appctx.CurrentContext{Cwd: "/tmp"}
	opts := intelligence.SearchOptions{
		Limit:  5,
		KBOnly: true,
	}

	results, err := intelligence.Search(store, "find big files over 100mb", ctx, opts)
	if err != nil {
		t.Fatalf("Search with KB failed: %v", err)
	}

	if len(results) == 0 {
		t.Error("expected at least one result from knowledge base")
		return
	}

	found := false
	for _, r := range results {
		if r.MatchType == "knowledge" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected at least one knowledge base result")
	}
}

func TestKnowledgeBaseLoad(t *testing.T) {
	store := tempVault(t)

	kbPath := findTestKBPath(t)
	if kbPath == "" {
		t.Skip("knowledge base not found, skipping")
	}

	count, err := intelligence.LoadKnowledgeBase(store, kbPath)
	if err != nil {
		t.Fatalf("LoadKnowledgeBase failed: %v", err)
	}
	if count < 100 {
		t.Errorf("expected at least 100 knowledge entries, got %d", count)
	}

	// Idempotent: second load should return existing count
	count2, err := intelligence.LoadKnowledgeBase(store, kbPath)
	if err != nil {
		t.Fatalf("second LoadKnowledgeBase failed: %v", err)
	}
	if count2 != count {
		t.Errorf("expected idempotent load to return %d, got %d", count, count2)
	}
}

func TestSessionTracking(t *testing.T) {
	session := appctx.GetSession("test-session-1")
	session.AddCommand("git status", 0, "git")
	session.AddCommand("git add .", 0, "git")
	session.AddCommand("docker ps", 0, "docker")

	recent := session.RecentCommands(2)
	if len(recent) != 2 {
		t.Errorf("expected 2 recent commands, got %d", len(recent))
	}
	if recent[0].Command != "git add ." {
		t.Errorf("expected second-to-last command 'git add .', got %q", recent[0].Command)
	}
	if recent[1].Command != "docker ps" {
		t.Errorf("expected last command 'docker ps', got %q", recent[1].Command)
	}

	cats := session.RecentCategories()
	if cats["git"] != 2 {
		t.Errorf("expected 2 git commands in session, got %d", cats["git"])
	}
	if cats["docker"] != 1 {
		t.Errorf("expected 1 docker command in session, got %d", cats["docker"])
	}
}

func TestProjectDetection(t *testing.T) {
	dir := t.TempDir()

	goDir := filepath.Join(dir, "goproject")
	os.MkdirAll(goDir, 0o755)
	os.WriteFile(filepath.Join(goDir, "go.mod"), []byte("module test"), 0o644)
	info := appctx.DetectProject(goDir)
	if info.Type != "go" {
		t.Errorf("expected project type 'go', got %q", info.Type)
	}

	nodeDir := filepath.Join(dir, "nodeproject")
	os.MkdirAll(nodeDir, 0o755)
	os.WriteFile(filepath.Join(nodeDir, "package.json"), []byte("{}"), 0o644)
	info = appctx.DetectProject(nodeDir)
	if info.Type != "node" {
		t.Errorf("expected project type 'node', got %q", info.Type)
	}

	emptyDir := filepath.Join(dir, "emptydir")
	os.MkdirAll(emptyDir, 0o755)
	info = appctx.DetectProject(emptyDir)
	if info.Type != "" {
		t.Errorf("expected empty project type for empty dir, got %q", info.Type)
	}
}

func TestGitDetection(t *testing.T) {
	// This test only verifies non-git directory behavior.
	dir := t.TempDir()
	state := appctx.DetectGit(dir)
	if state.IsRepo {
		t.Error("temp dir should not be detected as a git repo")
	}
}

func TestKnowledgeFTS5Search(t *testing.T) {
	store := tempVault(t)

	kbPath := findTestKBPath(t)
	if kbPath == "" {
		t.Skip("knowledge base not found, skipping")
	}

	_, err := intelligence.LoadKnowledgeBase(store, kbPath)
	if err != nil {
		t.Fatalf("LoadKnowledgeBase failed: %v", err)
	}

	results, err := store.SearchKnowledgeFTS5("docker OR container OR compose", 10)
	if err != nil {
		t.Fatalf("SearchKnowledgeFTS5 failed: %v", err)
	}
	if len(results) == 0 {
		t.Error("expected docker-related knowledge results")
	}
}

func TestContextsForCommand(t *testing.T) {
	store := tempVault(t)
	now := time.Now().UTC()

	cmd := &vault.Command{
		Raw: "go test ./...", Binary: "go", Category: "package",
		Flags: "[]", FirstSeen: now, LastSeen: now,
	}
	id, err := store.InsertCommand(cmd)
	if err != nil {
		t.Fatalf("InsertCommand failed: %v", err)
	}

	exitCode := 0
	for i := 0; i < 3; i++ {
		ctx := &vault.Context{
			CommandID:   id,
			Cwd:         "/home/user/project",
			GitRepo:     "recall",
			ProjectType: "go",
			Timestamp:   now.Add(time.Duration(i) * time.Hour),
			ExitCode:    &exitCode,
		}
		if err := store.InsertContext(ctx); err != nil {
			t.Fatalf("InsertContext failed: %v", err)
		}
	}

	contexts, err := store.GetContextsForCommand(id)
	if err != nil {
		t.Fatalf("GetContextsForCommand failed: %v", err)
	}
	if len(contexts) != 3 {
		t.Errorf("expected 3 contexts, got %d", len(contexts))
	}
}

func TestMaxFrequency(t *testing.T) {
	store := tempVault(t)
	now := time.Now().UTC()

	cmds := []vault.Command{
		{Raw: "git status", Binary: "git", Frequency: 50, Flags: "[]", FirstSeen: now, LastSeen: now},
		{Raw: "docker ps", Binary: "docker", Frequency: 10, Flags: "[]", FirstSeen: now, LastSeen: now},
	}
	store.BatchInsertCommands(cmds)

	maxFreq, err := store.GetMaxFrequency()
	if err != nil {
		t.Fatalf("GetMaxFrequency failed: %v", err)
	}
	if maxFreq < 10 {
		t.Errorf("expected max frequency >= 10, got %d", maxFreq)
	}
}

func findTestKBPath(t *testing.T) string {
	t.Helper()
	candidates := []string{
		"../data/knowledge.json",
		"data/knowledge.json",
		filepath.Join("..", "data", "knowledge.json"),
	}

	cwd, _ := os.Getwd()
	for _, c := range candidates {
		path := filepath.Join(cwd, c)
		if _, err := os.Stat(path); err == nil {
			return path
		}
	}

	absPath := filepath.Join(filepath.Dir(cwd), "data", "knowledge.json")
	if _, err := os.Stat(absPath); err == nil {
		return absPath
	}

	return ""
}

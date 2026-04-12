package tests

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Om-Rohilla/recall/internal/capture"
	"github.com/Om-Rohilla/recall/internal/vault"
	"github.com/Om-Rohilla/recall/pkg/config"
	"github.com/Om-Rohilla/recall/pkg/shell"
)

// --- AES-256-GCM Encryption Tests ---

func TestEncryptDecryptRoundtrip(t *testing.T) {
	tests := []struct {
		name      string
		plaintext string
	}{
		{"short text", "hello world"},
		{"empty", ""},
		{"long text", strings.Repeat("recall is great! ", 1000)},
		{"binary-like", "\x00\x01\x02\xff\xfe\xfd"},
		{"unicode", "日本語テスト 🚀 émojis"},
		{"json payload", `{"commands":[{"raw":"git push","frequency":42}]}`},
	}

	key, err := vault.DeriveKey("test-password-123", []byte("fixed-salt-for-testing-32bytes!!"))
	if err != nil {
		t.Fatalf("DeriveKey: %v", err)
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			encrypted, err := vault.Encrypt([]byte(tt.plaintext), key)
			if err != nil {
				t.Fatalf("encrypt failed: %v", err)
			}

			if tt.plaintext != "" && bytes.Equal(encrypted, []byte(tt.plaintext)) {
				t.Fatal("encrypted data should differ from plaintext")
			}

			decrypted, err := vault.Decrypt(encrypted, key)
			if err != nil {
				t.Fatalf("decrypt failed: %v", err)
			}

			if string(decrypted) != tt.plaintext {
				t.Fatalf("roundtrip mismatch: got %q, want %q", string(decrypted), tt.plaintext)
			}
		})
	}
}

func TestArgon2idConsistentKeys(t *testing.T) {
	password := "my-secure-password"
	salt := []byte("consistent-salt-32-bytes-long!!!!")

	key1, err := vault.DeriveKey(password, salt)
	if err != nil {
		t.Fatalf("DeriveKey: %v", err)
	}
	key2, err := vault.DeriveKey(password, salt)
	if err != nil {
		t.Fatalf("DeriveKey: %v", err)
	}

	if !bytes.Equal(key1, key2) {
		t.Fatal("same password + salt should produce identical keys")
	}

	if len(key1) != 32 {
		t.Fatalf("key length should be 32 bytes, got %d", len(key1))
	}
}

func TestDifferentPasswordsDifferentKeys(t *testing.T) {
	salt := []byte("same-salt-for-both-tests-32bytes")

	key1, _ := vault.DeriveKey("password-one", salt)
	key2, _ := vault.DeriveKey("password-two", salt)

	if bytes.Equal(key1, key2) {
		t.Fatal("different passwords should produce different keys")
	}
}

func TestDifferentSaltsDifferentKeys(t *testing.T) {
	password := "same-password"

	salt1 := []byte("salt-one-is-32-bytes-long-!!!!!!")
	salt2 := []byte("salt-two-is-32-bytes-long-!!!!!!")

	key1, _ := vault.DeriveKey(password, salt1)
	key2, _ := vault.DeriveKey(password, salt2)

	if bytes.Equal(key1, key2) {
		t.Fatal("different salts should produce different keys")
	}
}

func TestTamperedCiphertextFails(t *testing.T) {
	key, _ := vault.DeriveKey("integrity-test", []byte("salt-for-integrity-test-32bytes!"))

	encrypted, err := vault.Encrypt([]byte("sensitive data"), key)
	if err != nil {
		t.Fatalf("encrypt failed: %v", err)
	}

	// Tamper with the ciphertext (flip a byte near the end)
	tampered := make([]byte, len(encrypted))
	copy(tampered, encrypted)
	tampered[len(tampered)-1] ^= 0xFF

	_, err = vault.Decrypt(tampered, key)
	if err == nil {
		t.Fatal("decryption of tampered ciphertext should fail")
	}
}

func TestWrongPasswordFails(t *testing.T) {
	salt := []byte("wrong-password-test-salt-32bytes!")
	correctKey, _ := vault.DeriveKey("correct-password", salt)
	wrongKey, _ := vault.DeriveKey("wrong-password", salt)

	encrypted, err := vault.Encrypt([]byte("secret message"), correctKey)
	if err != nil {
		t.Fatalf("encrypt failed: %v", err)
	}

	_, err = vault.Decrypt(encrypted, wrongKey)
	if err == nil {
		t.Fatal("decryption with wrong password should fail")
	}
}

func TestGenerateSaltUniqueness(t *testing.T) {
	salt1, err := vault.GenerateSalt()
	if err != nil {
		t.Fatalf("generate salt 1: %v", err)
	}
	salt2, err := vault.GenerateSalt()
	if err != nil {
		t.Fatalf("generate salt 2: %v", err)
	}

	if bytes.Equal(salt1, salt2) {
		t.Fatal("two generated salts should not be identical")
	}
	if len(salt1) != vault.SaltSize {
		t.Fatalf("salt size should be %d, got %d", vault.SaltSize, len(salt1))
	}
}

// --- Export File Format Tests ---

func TestPackUnpackExportRoundtrip(t *testing.T) {
	salt, _ := vault.GenerateSalt()
	encrypted := []byte("this is some encrypted data for testing the format")

	packed := vault.PackExport(salt, encrypted)

	gotSalt, gotData, err := vault.UnpackExport(packed)
	if err != nil {
		t.Fatalf("unpack failed: %v", err)
	}

	if !bytes.Equal(gotSalt, salt) {
		t.Fatal("salt mismatch after pack/unpack")
	}
	if !bytes.Equal(gotData, encrypted) {
		t.Fatal("encrypted data mismatch after pack/unpack")
	}
}

func TestUnpackInvalidFormat(t *testing.T) {
	tests := []struct {
		name string
		data []byte
	}{
		{"empty", []byte{}},
		{"too short", []byte("REC")},
		{"wrong magic", []byte("WRNG\x01" + strings.Repeat("a", 32) + "data")},
		{"wrong version", []byte("RECL\x99" + strings.Repeat("a", 32) + "data")},
		{"no data after header", []byte("RECL\x01" + strings.Repeat("a", 32))},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, _, err := vault.UnpackExport(tt.data)
			if err == nil {
				t.Fatal("should reject invalid format")
			}
		})
	}
}

// --- Export/Import Vault Roundtrip Tests ---

func TestExportImportVaultRoundtrip(t *testing.T) {
	dir := t.TempDir()

	// Create and populate source vault
	srcPath := filepath.Join(dir, "source.db")
	src, err := vault.NewStore(srcPath)
	if err != nil {
		t.Fatalf("creating source vault: %v", err)
	}

	exitCode := 0
	now := time.Now().UTC()
	cmds := []vault.Command{
		{Raw: "git push origin main", Binary: "git", Subcommand: "push", Category: "git", Frequency: 15, FirstSeen: now, LastSeen: now, LastExit: &exitCode},
		{Raw: "docker compose up -d", Binary: "docker", Subcommand: "compose up", Category: "docker", Frequency: 8, FirstSeen: now, LastSeen: now, LastExit: &exitCode},
		{Raw: "find . -name '*.go' -type f", Binary: "find", Category: "filesystem", Frequency: 3, FirstSeen: now, LastSeen: now},
	}

	for i := range cmds {
		cmdID, err := src.InsertCommand(&cmds[i])
		if err != nil {
			t.Fatalf("inserting command: %v", err)
		}
		_ = src.InsertContext(&vault.Context{
			CommandID: cmdID,
			Cwd:       "/home/user/project",
			GitRepo:   "myproject",
			GitBranch: "main",
			Timestamp: now,
		})
	}

	// Export
	exportData, err := src.ExportVaultData(false)
	if err != nil {
		t.Fatalf("exporting vault: %v", err)
	}
	src.Close()

	if exportData.CommandCount != 3 {
		t.Fatalf("expected 3 commands in export, got %d", exportData.CommandCount)
	}
	if exportData.ContextCount != 3 {
		t.Fatalf("expected 3 contexts in export, got %d", exportData.ContextCount)
	}

	// Serialize, encrypt, decrypt, deserialize (full roundtrip)
	jsonData, err := json.Marshal(exportData)
	if err != nil {
		t.Fatalf("marshaling export: %v", err)
	}

	salt, _ := vault.GenerateSalt()
	key, _ := vault.DeriveKey("export-test-password", salt)

	encrypted, err := vault.Encrypt(jsonData, key)
	if err != nil {
		t.Fatalf("encrypting: %v", err)
	}

	packed := vault.PackExport(salt, encrypted)

	gotSalt, gotEncData, err := vault.UnpackExport(packed)
	if err != nil {
		t.Fatalf("unpacking: %v", err)
	}

	gotKey, _ := vault.DeriveKey("export-test-password", gotSalt)
	decrypted, err := vault.Decrypt(gotEncData, gotKey)
	if err != nil {
		t.Fatalf("decrypting: %v", err)
	}

	var importedData vault.ExportData
	if err := json.Unmarshal(decrypted, &importedData); err != nil {
		t.Fatalf("unmarshaling import: %v", err)
	}

	dstPath := filepath.Join(dir, "dest.db")
	dst, err := vault.NewStore(dstPath)
	if err != nil {
		t.Fatalf("creating dest vault: %v", err)
	}
	defer dst.Close()

	importedCmds, importedCtxs, err := dst.ImportVaultData(&importedData, false)
	if err != nil {
		t.Fatalf("importing: %v", err)
	}

	if importedCmds != 3 {
		t.Fatalf("expected 3 imported commands, got %d", importedCmds)
	}
	if importedCtxs != 3 {
		t.Fatalf("expected 3 imported contexts, got %d", importedCtxs)
	}

	// Verify data integrity
	stats, _ := dst.GetStats()
	if stats.UniqueCommands != 3 {
		t.Fatalf("expected 3 unique commands in dest, got %d", stats.UniqueCommands)
	}
}

func TestPlainExportProducesValidJSON(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "test.db")

	store, err := vault.NewStore(dbPath)
	if err != nil {
		t.Fatalf("creating vault: %v", err)
	}
	defer store.Close()

	now := time.Now().UTC()
	_, _ = store.InsertCommand(&vault.Command{
		Raw: "echo hello", Binary: "echo", Category: "other", Frequency: 1, FirstSeen: now, LastSeen: now,
	})

	data, err := store.ExportVaultData(false)
	if err != nil {
		t.Fatalf("exporting: %v", err)
	}

	jsonData, err := json.Marshal(data)
	if err != nil {
		t.Fatalf("marshaling: %v", err)
	}

	// Verify it's valid JSON
	var parsed vault.ExportData
	if err := json.Unmarshal(jsonData, &parsed); err != nil {
		t.Fatalf("plain export is not valid JSON: %v", err)
	}

	if parsed.Version != 1 {
		t.Fatalf("expected version 1, got %d", parsed.Version)
	}
	if parsed.CommandCount != 1 {
		t.Fatalf("expected 1 command, got %d", parsed.CommandCount)
	}
}

func TestCommandsOnlyExport(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "test.db")

	store, err := vault.NewStore(dbPath)
	if err != nil {
		t.Fatalf("creating vault: %v", err)
	}
	defer store.Close()

	now := time.Now().UTC()
	cmdID, _ := store.InsertCommand(&vault.Command{
		Raw: "ls -la", Binary: "ls", Category: "filesystem", Frequency: 1, FirstSeen: now, LastSeen: now,
	})
	_ = store.InsertContext(&vault.Context{
		CommandID: cmdID, Cwd: "/tmp", Timestamp: now,
	})

	data, err := store.ExportVaultData(true)
	if err != nil {
		t.Fatalf("exporting: %v", err)
	}

	if len(data.Commands) != 1 {
		t.Fatalf("expected 1 command, got %d", len(data.Commands))
	}
	if len(data.Contexts) != 0 {
		t.Fatalf("commands-only export should have 0 contexts, got %d", len(data.Contexts))
	}
}

func TestMergeImportBehavior(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "merge.db")

	store, err := vault.NewStore(dbPath)
	if err != nil {
		t.Fatalf("creating vault: %v", err)
	}
	defer store.Close()

	now := time.Now().UTC()

	// Insert existing command
	_, _ = store.InsertCommand(&vault.Command{
		Raw: "git status", Binary: "git", Subcommand: "status", Category: "git", Frequency: 5, FirstSeen: now, LastSeen: now,
	})
	_, _ = store.InsertCommand(&vault.Command{
		Raw: "make build", Binary: "make", Subcommand: "build", Category: "build", Frequency: 2, FirstSeen: now, LastSeen: now,
	})

	// Import data with an overlapping command and a new one
	importData := &vault.ExportData{
		Version:    1,
		ExportedAt: now,
		Commands: []vault.Command{
			{ID: 100, Raw: "git status", Binary: "git", Subcommand: "status", Category: "git", Frequency: 10, FirstSeen: now, LastSeen: now},
			{ID: 200, Raw: "docker ps", Binary: "docker", Subcommand: "ps", Category: "docker", Frequency: 3, FirstSeen: now, LastSeen: now},
		},
		CommandCount: 2,
	}

	importedCmds, _, err := store.ImportVaultData(importData, true)
	if err != nil {
		t.Fatalf("merge import: %v", err)
	}

	if importedCmds != 2 {
		t.Fatalf("expected 2 imported commands (1 merged + 1 new), got %d", importedCmds)
	}

	// Verify "git status" frequency was incremented
	allCmds, _ := store.GetAllCommands("frequency", 0)
	for _, cmd := range allCmds {
		if cmd.Raw == "git status" && cmd.Frequency < 6 {
			t.Fatalf("git status frequency should be >= 6 after merge, got %d", cmd.Frequency)
		}
	}

	// Verify all 3 commands exist
	stats, _ := store.GetStats()
	if stats.UniqueCommands != 3 {
		t.Fatalf("expected 3 unique commands after merge, got %d", stats.UniqueCommands)
	}
}

// --- Config Tests ---

func TestConfigSetGetReset(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.toml")
	t.Setenv("RECALL_CONFIG", configPath)

	config.Reset()
	cfg := config.DefaultConfig()

	// Save defaults
	if err := config.Save(cfg); err != nil {
		t.Fatalf("saving default config: %v", err)
	}

	// Verify defaults
	if cfg.Search.TopResults != 1 {
		t.Fatalf("expected default top_results=1, got %d", cfg.Search.TopResults)
	}

	// Modify and save
	cfg.Search.TopResults = 5
	cfg.Search.MinConfidence = 0.5
	cfg.Alias.MinFrequency = 10
	if err := config.Save(cfg); err != nil {
		t.Fatalf("saving modified config: %v", err)
	}

	// Reload and verify
	config.Reset()
	reloaded, err := config.Load()
	if err != nil {
		t.Fatalf("reloading config: %v", err)
	}

	if reloaded.Search.TopResults != 5 {
		t.Fatalf("expected top_results=5 after reload, got %d", reloaded.Search.TopResults)
	}
	if reloaded.Search.MinConfidence != 0.5 {
		t.Fatalf("expected min_confidence=0.5 after reload, got %f", reloaded.Search.MinConfidence)
	}
	if reloaded.Alias.MinFrequency != 10 {
		t.Fatalf("expected min_frequency=10 after reload, got %d", reloaded.Alias.MinFrequency)
	}

	// Reset to defaults
	defaultCfg := config.DefaultConfig()
	if err := config.Save(defaultCfg); err != nil {
		t.Fatalf("saving reset config: %v", err)
	}
	config.Reset()
	resetCfg, _ := config.Load()
	if resetCfg.Search.TopResults != 1 {
		t.Fatalf("expected top_results=1 after reset, got %d", resetCfg.Search.TopResults)
	}
}

func TestConfigDefaultPath(t *testing.T) {
	path := config.DefaultConfigPath()
	if path == "" {
		t.Fatal("default config path should not be empty")
	}
	if !strings.Contains(path, "recall") {
		t.Fatalf("config path should contain 'recall', got %s", path)
	}
}

// --- Fish Hook Tests ---

func TestFishHookScriptGeneration(t *testing.T) {
	script, err := shell.HookScript(shell.Fish)
	if err != nil {
		t.Fatalf("generating fish hook: %v", err)
	}

	requiredParts := []string{
		"fish_postexec",
		"recall capture",
		"--command",
		"--exit-code",
		"--cwd",
		"--duration",
		"--session",
		"RECALL_SESSION_ID",
		"CMD_DURATION",
		"__recall_search",
		"__recall_vault",
		"__recall_explain",
		"bind",
	}

	for _, part := range requiredParts {
		if !strings.Contains(script, part) {
			t.Errorf("fish hook script missing required part: %q", part)
		}
	}
}

func TestFishHookHasHotkeyBindings(t *testing.T) {
	script, err := shell.HookScript(shell.Fish)
	if err != nil {
		t.Fatalf("generating fish hook: %v", err)
	}

	// Ctrl+Space — inline search (unchanged)
	if !strings.Contains(script, `bind \c@`) {
		t.Error("fish hook missing Ctrl+Space binding")
	}
	// Alt+K — vault browser (replaced old Ctrl+K which conflicted with readline kill-line)
	if !strings.Contains(script, `bind \ek`) {
		t.Error("fish hook missing Alt+K (vault) binding — Ctrl+K was removed due to readline conflict")
	}
	// Alt+R — explain (replaced old Ctrl+E which conflicted with readline end-of-line)
	if !strings.Contains(script, `bind \er`) {
		t.Error("fish hook missing Alt+R (explain) binding — Ctrl+E was removed due to readline conflict")
	}
}

func TestFishHookCapturesDuration(t *testing.T) {
	script, err := shell.HookScript(shell.Fish)
	if err != nil {
		t.Fatalf("generating fish hook: %v", err)
	}

	if !strings.Contains(script, "CMD_DURATION") {
		t.Error("fish hook should use CMD_DURATION for duration measurement")
	}
}

func TestFishShellInfo(t *testing.T) {
	info, err := shell.ForShell("fish")
	if err != nil {
		t.Fatalf("ForShell(fish): %v", err)
	}

	if info.Shell != shell.Fish {
		t.Fatalf("expected shell=fish, got %s", info.Shell)
	}
	if !strings.Contains(info.ConfigPath, "config.fish") {
		t.Errorf("expected config path to contain config.fish, got %s", info.ConfigPath)
	}
}

// --- Export File End-to-End Test ---

func TestEncryptedExportFile(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "test.db")
	exportPath := filepath.Join(dir, "vault.enc")

	store, err := vault.NewStore(dbPath)
	if err != nil {
		t.Fatalf("creating vault: %v", err)
	}

	now := time.Now().UTC()
	for i := 0; i < 5; i++ {
		_, _ = store.InsertCommand(&vault.Command{
			Raw: "cmd-" + string(rune('a'+i)), Binary: "cmd", Category: "test", Frequency: i + 1, FirstSeen: now, LastSeen: now,
		})
	}

	data, _ := store.ExportVaultData(false)
	store.Close()

	jsonData, _ := json.Marshal(data)

	password := "test-export-password"
	salt, _ := vault.GenerateSalt()
	key, _ := vault.DeriveKey(password, salt)
	encrypted, _ := vault.Encrypt(jsonData, key)
	packed := vault.PackExport(salt, encrypted)

	if err := os.WriteFile(exportPath, packed, 0o600); err != nil {
		t.Fatalf("writing export file: %v", err)
	}

	fileData, _ := os.ReadFile(exportPath)

	gotSalt, gotEncData, err := vault.UnpackExport(fileData)
	if err != nil {
		t.Fatalf("unpacking file: %v", err)
	}

	gotKey, _ := vault.DeriveKey(password, gotSalt)
	decrypted, err := vault.Decrypt(gotEncData, gotKey)
	if err != nil {
		t.Fatalf("decrypting file: %v", err)
	}

	var result vault.ExportData
	if err := json.Unmarshal(decrypted, &result); err != nil {
		t.Fatalf("unmarshaling: %v", err)
	}

	if result.CommandCount != 5 {
		t.Fatalf("expected 5 commands, got %d", result.CommandCount)
	}
}

// --- Encrypted Vault at Rest Tests ---

func TestEncryptedVaultRoundtrip(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "vault.db")

	// Pin a deterministic 32-byte vault key for the duration of this test.
	// Without this, two consecutive NewStore calls generate different keys and
	// the second open fails with "file is not a database".
	keyBytes := make([]byte, 32)
	for i := range keyBytes {
		keyBytes[i] = byte(i + 1)
	}
	keyHex := fmt.Sprintf("%x", keyBytes)
	t.Setenv("RECALL_VAULT_KEY", keyHex)

	store, err := vault.NewStore(dbPath)
	if err != nil {
		t.Fatalf("creating encrypted vault: %v", err)
	}

	now := time.Now().UTC()
	_, err = store.InsertCommand(&vault.Command{
		Raw: "git push origin main", Binary: "git", Subcommand: "push",
		Category: "git", Frequency: 1, FirstSeen: now, LastSeen: now,
	})
	if err != nil {
		t.Fatalf("inserting command: %v", err)
	}

	if err := store.Close(); err != nil {
		t.Fatalf("closing encrypted vault: %v", err)
	}

	// SQLCipher encrypts the database IN-PLACE — vault.db is the encrypted file.
	// There is no .enc sidecar. Verify the db exists and is not plaintext.
	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		t.Fatal("encrypted vault.db should exist after close")
	}
	rawData, _ := os.ReadFile(dbPath)
	if strings.Contains(string(rawData), "git push") {
		t.Fatal("vault.db should not contain plaintext commands — SQLCipher encryption failed")
	}

	store2, err := vault.NewStore(dbPath)
	if err != nil {
		t.Fatalf("reopening encrypted vault: %v", err)
	}
	defer store2.Close()

	stats, err := store2.GetStats()
	if err != nil {
		t.Fatalf("getting stats: %v", err)
	}
	if stats.UniqueCommands != 1 {
		t.Fatalf("expected 1 command after decrypt, got %d", stats.UniqueCommands)
	}
}

func TestVaultDirectoryPermissions(t *testing.T) {
	dir := t.TempDir()
	vaultDir := filepath.Join(dir, "recall_vault")
	dbPath := filepath.Join(vaultDir, "vault.db")

	store, err := vault.NewStore(dbPath)
	if err != nil {
		t.Fatalf("creating vault: %v", err)
	}
	defer store.Close()

	info, err := os.Stat(vaultDir)
	if err != nil {
		t.Fatalf("stat vault dir: %v", err)
	}

	perm := info.Mode().Perm()
	if perm&0o077 != 0 {
		t.Errorf("vault directory should be 0700, got %o", perm)
	}
}

func TestFTSQuerySanitizationBlocksWildcard(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "test.db")
	store, err := vault.NewStore(dbPath)
	if err != nil {
		t.Fatalf("creating vault: %v", err)
	}
	defer store.Close()

	now := time.Now().UTC()
	_, _ = store.InsertCommand(&vault.Command{
		Raw: "git status", Binary: "git", Category: "git",
		Frequency: 1, FirstSeen: now, LastSeen: now, Flags: "[]",
	})
	_ = store.RebuildFTSIndex()

	results, err := store.SearchFTS5("git*", 10)
	if err != nil {
		t.Fatalf("FTS5 search with wildcard: %v", err)
	}
	if len(results) > 0 {
		for _, r := range results {
			if strings.Contains(r.Command.Raw, "*") {
				t.Error("wildcard should not appear in results")
			}
		}
	}
}

func TestSchemaMigration(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "migrate.db")

	// Pin a consistent vault key so both NewStore calls open with the same
	// SQLCipher key. Without this, GetOrGenerateVaultKey() generates a fresh
	// random key on each call (keyring unavailable in test env), causing the
	// second open to fail with "file is not a database".
	t.Setenv("RECALL_VAULT_KEY",
		"0102030405060708090a0b0c0d0e0f101112131415161718191a1b1c1d1e1f20")

	store, err := vault.NewStore(dbPath)
	if err != nil {
		t.Fatalf("creating vault: %v", err)
	}
	store.Close()

	store2, err := vault.NewStore(dbPath)
	if err != nil {
		t.Fatalf("reopening vault (migration): %v", err)
	}
	defer store2.Close()

	stats, err := store2.GetStats()
	if err != nil {
		t.Fatalf("getting stats after migration: %v", err)
	}
	if stats == nil {
		t.Fatal("stats should not be nil")
	}
}

// --- Enhanced Secret Filtering Tests ---

func TestSecretFilteringEnhanced(t *testing.T) {
	cfg := config.DefaultConfig()

	tests := []struct {
		name    string
		cmd     string
		allowed bool
	}{
		{"bearer token", "curl -H 'Authorization: Bearer abc123'", false},
		{"github token", "export GITHUB_TOKEN=ghp_abc123", false},
		{"openai key", "export OPENAI_API_KEY=sk-abc123", false},
		{"private key inline", "echo '-----BEGIN RSA PRIVATE KEY-----'", false},
		{"mysql password", "mysql -pMySecret123", false},
		{"mongodb uri", "mongosh mongodb+srv://user:pass@cluster.net", false},
		{"connection string", "psql://admin:secret@localhost/db", false},
		{"safe command", "git commit -m 'fix typo'", true},
		{"safe docker", "docker compose up -d", true},
		// --- 2D: New patterns ---
		{"fly.io fo1_ token", "flyctl auth token fo1_abcdef1234567890", false},
		{"fly.io fo2_ token", "export FLY_API_TOKEN=fo2_abcdef1234567890", false},
		{"sentry dsn eq", "export sentry_dsn=https://key@sentry.io/123", false},
		{"sentry dsn dot", "sentry.dsn=https://key@sentry.io/123 myapp", false},
		{"vercel token", "vercel_token=abc123secret deploy", false},
		{"1password op uri", "op read op://vault/item/field", false},
		{"doppler ct token", "export DOPPLER_TOKEN=dp.ct.abc123def456", false},
		{"doppler pt token", "export DOPPLER_TOKEN=dp.pt.abc123def456", false},
		{"doppler st token", "export DOPPLER_TOKEN=dp.st.abc123def456", false},
		{"hashicorp vault hvs", "vault token lookup hvs.abc123def456ghi789", false},
		{"hashicorp vault hvb", "export VAULT_TOKEN=hvb.abc123def456ghi789", false},
		{"hashicorp vault hvr", "export VAULT_TOKEN=hvr.abc123def456ghi789", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := capture.Filter(tt.cmd, cfg)
			if result.Allowed != tt.allowed {
				t.Errorf("Filter(%q) = allowed:%v, want %v (reason: %s)",
					tt.cmd, result.Allowed, tt.allowed, result.Reason)
			}
		})
	}
}

// --- Additional Store Method Tests ---

func TestGetAllContexts(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "test.db")

	store, err := vault.NewStore(dbPath)
	if err != nil {
		t.Fatalf("creating vault: %v", err)
	}
	defer store.Close()

	now := time.Now().UTC()
	cmdID, _ := store.InsertCommand(&vault.Command{
		Raw: "test cmd", Binary: "test", Category: "test", Frequency: 1, FirstSeen: now, LastSeen: now,
	})

	_ = store.InsertContext(&vault.Context{CommandID: cmdID, Cwd: "/home", Timestamp: now})
	_ = store.InsertContext(&vault.Context{CommandID: cmdID, Cwd: "/tmp", Timestamp: now})

	ctxs, err := store.GetAllContexts()
	if err != nil {
		t.Fatalf("GetAllContexts: %v", err)
	}
	if len(ctxs) != 2 {
		t.Fatalf("expected 2 contexts, got %d", len(ctxs))
	}
}

func TestGetAllPatterns(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "test.db")

	store, err := vault.NewStore(dbPath)
	if err != nil {
		t.Fatalf("creating vault: %v", err)
	}
	defer store.Close()

	_, _ = store.DB().Exec(
		"INSERT INTO patterns (template, frequency, suggested_alias) VALUES (?, ?, ?)",
		"git commit -m {arg}", 10, "gc",
	)

	patterns, err := store.GetAllPatterns()
	if err != nil {
		t.Fatalf("GetAllPatterns: %v", err)
	}
	if len(patterns) != 1 {
		t.Fatalf("expected 1 pattern, got %d", len(patterns))
	}
	if patterns[0].Template != "git commit -m {arg}" {
		t.Fatalf("unexpected template: %s", patterns[0].Template)
	}
}

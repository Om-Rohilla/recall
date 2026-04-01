package intelligence

import (
	"testing"
)

func TestExtractIntent_SimpleQuery(t *testing.T) {
	intent := ExtractIntent("find large files")
	if intent.RawQuery != "find large files" {
		t.Errorf("expected raw query 'find large files', got %q", intent.RawQuery)
	}
	if len(intent.Tokens) == 0 {
		t.Error("expected tokens to be extracted")
	}
}

func TestExtractIntent_EmptyQuery(t *testing.T) {
	intent := ExtractIntent("")
	if len(intent.Tokens) != 0 {
		t.Errorf("empty query should produce no tokens, got %v", intent.Tokens)
	}
}

func TestExtractIntent_CategoryDetection(t *testing.T) {
	tests := []struct {
		query    string
		category string
	}{
		{"git commit changes", "git"},
		{"docker container list", "docker"},
		{"kubernetes pod status", "kubernetes"},
		{"find files in directory", "filesystem"},
		{"compress archive files", "archive"},
	}

	for _, tt := range tests {
		t.Run(tt.query, func(t *testing.T) {
			intent := ExtractIntent(tt.query)
			if intent.Category != tt.category {
				t.Errorf("expected category %q, got %q", tt.category, intent.Category)
			}
		})
	}
}

func TestExtractIntent_ActionDetection(t *testing.T) {
	tests := []struct {
		query  string
		action string
	}{
		{"delete old logs", "delete"},
		{"find large files", "find"},
		{"install new package", "install"},
		{"show disk usage", "show"},
	}

	for _, tt := range tests {
		t.Run(tt.query, func(t *testing.T) {
			intent := ExtractIntent(tt.query)
			if intent.Action != tt.action {
				t.Errorf("expected action %q, got %q", tt.action, intent.Action)
			}
		})
	}
}

func TestExpandSynonyms_KnownTokens(t *testing.T) {
	// Only test tokens we know are in synonymDict
	tests := []struct {
		token    string
		expanded bool
	}{
		{"remove", true},    // → delete, rm, erase, purge
		{"folder", true},    // → directory, dir
		{"container", true}, // → containers
		{"find", true},      // → search, locate, look
		{"delete", true},    // → remove, rm, erase, purge
		{"normalword", false},
	}

	for _, tt := range tests {
		t.Run(tt.token, func(t *testing.T) {
			result := expandSynonyms([]string{tt.token})
			hasExpansion := len(result) > 1
			if hasExpansion != tt.expanded {
				t.Errorf("token %q: expected expanded=%v, got %v (result: %v)", tt.token, tt.expanded, hasExpansion, result)
			}
		})
	}
}

func TestBuildFTSQuery_Basic(t *testing.T) {
	intent := ExtractIntent("find large files")
	query := BuildFTSQuery(intent)
	if query == "" {
		t.Error("FTS query should not be empty")
	}
}

func TestBuildFTSQuery_Empty(t *testing.T) {
	intent := ExtractIntent("")
	query := BuildFTSQuery(intent)
	// Empty intent might return empty query
	_ = query
}

func TestBuildFTSQuery_SpecialCharacters(t *testing.T) {
	queries := []string{
		"find -name '*.go'",
		"grep -E 'foo|bar'",
		`echo "hello world"`,
		"test && pass",
		"cmd1 || cmd2",
	}
	for _, q := range queries {
		t.Run(q, func(t *testing.T) {
			intent := ExtractIntent(q)
			result := BuildFTSQuery(intent)
			_ = result // just verify no panic
		})
	}
}

func TestDetectAction_ValidActions(t *testing.T) {
	tests := []struct {
		tokens []string
		action string
	}{
		{[]string{"find", "files"}, "find"},
		{[]string{"delete", "old", "logs"}, "delete"},
		{[]string{"show", "disk", "usage"}, "show"},
		{[]string{"install", "package"}, "install"},
		{[]string{"unknown", "stuff"}, ""},
	}

	for _, tt := range tests {
		t.Run(tt.action, func(t *testing.T) {
			result := detectAction(tt.tokens)
			if result != tt.action {
				t.Errorf("expected action %q, got %q", tt.action, result)
			}
		})
	}
}

func TestDetectTarget_ValidTargets(t *testing.T) {
	tests := []struct {
		tokens []string
		target string
	}{
		{[]string{"find", "file"}, "file"},
		{[]string{"show", "directory"}, "directory"},
		{[]string{"kill", "process"}, "process"},
		{[]string{"check", "port"}, "port"},
		{[]string{"unknown"}, ""},
	}

	for _, tt := range tests {
		t.Run(tt.target, func(t *testing.T) {
			result := detectTarget(tt.tokens)
			if result != tt.target {
				t.Errorf("expected target %q, got %q", tt.target, result)
			}
		})
	}
}

func TestDetectQueryCategory(t *testing.T) {
	tests := []struct {
		tokens   []string
		category string
	}{
		{[]string{"git", "commit"}, "git"},
		{[]string{"docker", "container"}, "docker"},
		{[]string{"kubectl", "pod"}, "kubernetes"},
		{[]string{"find", "file", "directory"}, "filesystem"},
	}

	for _, tt := range tests {
		t.Run(tt.category, func(t *testing.T) {
			result := detectQueryCategory(tt.tokens)
			if result != tt.category {
				t.Errorf("expected category %q, got %q", tt.category, result)
			}
		})
	}
}

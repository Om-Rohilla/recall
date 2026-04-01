package capture

import (
	"testing"
)

func TestParse_EmptyString(t *testing.T) {
	result := Parse("")
	if result.Binary != "" {
		t.Errorf("empty input should yield empty binary, got %q", result.Binary)
	}
}

func TestParse_SimpleCommand(t *testing.T) {
	result := Parse("git status")
	if result.Binary != "git" {
		t.Errorf("expected binary 'git', got %q", result.Binary)
	}
	if result.Subcommand != "status" {
		t.Errorf("expected subcommand 'status', got %q", result.Subcommand)
	}
	if result.Category != "git" {
		t.Errorf("expected category 'git', got %q", result.Category)
	}
}

func TestParse_CommandWithFlags(t *testing.T) {
	result := Parse("grep -rn 'TODO' .")
	if result.Binary != "grep" {
		t.Errorf("expected binary 'grep', got %q", result.Binary)
	}
	if len(result.Flags) < 1 {
		t.Error("expected flags to be parsed")
	}
	if result.Category != "text" {
		t.Errorf("expected category 'text', got %q", result.Category)
	}
}

func TestParse_SudoPrefix(t *testing.T) {
	result := Parse("sudo apt install vim")
	// sudo is stripped, apt is binary; apt is a multi-cmd tool so "install" and "vim" are subcommands
	if result.Binary != "apt" {
		t.Errorf("expected binary 'apt' (sudo stripped), got %q", result.Binary)
	}
	// apt is in multiCmdTools, so "install" and "vim" are both subcommands
	if result.Subcommand == "" {
		t.Error("expected a non-empty subcommand")
	}
}

func TestParse_EnvVarPrefix(t *testing.T) {
	result := Parse("NODE_ENV=production node server.js")
	if result.Binary != "node" {
		t.Errorf("expected binary 'node' (env stripped), got %q", result.Binary)
	}
}

func TestParse_MultipleEnvPrefixes(t *testing.T) {
	result := Parse("CC=gcc CXX=g++ make build")
	if result.Binary != "make" {
		t.Errorf("expected binary 'make', got %q", result.Binary)
	}
}

func TestParse_PipedCommand(t *testing.T) {
	result := Parse("cat file.txt | grep 'error' | wc -l")
	if result.Binary != "cat" {
		t.Errorf("expected first segment binary 'cat', got %q", result.Binary)
	}
	if result.Raw != "cat file.txt | grep 'error' | wc -l" {
		t.Errorf("raw should preserve full pipe chain, got %q", result.Raw)
	}
}

func TestParse_ChainedCommand(t *testing.T) {
	result := Parse("make build && make test")
	if result.Binary != "make" {
		t.Errorf("expected first segment binary 'make', got %q", result.Binary)
	}
	// make is NOT in multiCmdTools, so 'build' is an arg, not subcommand
	if result.Raw != "make build && make test" {
		t.Errorf("expected full raw preserved, got %q", result.Raw)
	}
}

func TestParse_SemicolonChain(t *testing.T) {
	result := Parse("echo hello; echo world")
	if result.Binary != "echo" {
		t.Errorf("expected first segment binary 'echo', got %q", result.Binary)
	}
}

func TestParse_QuotedArguments(t *testing.T) {
	result := Parse(`grep -r "hello world" .`)
	if result.Binary != "grep" {
		t.Errorf("expected binary 'grep', got %q", result.Binary)
	}
}

func TestParse_PipeInsideQuotes(t *testing.T) {
	result := Parse(`echo "hello | world"`)
	if result.Binary != "echo" {
		t.Errorf("expected binary 'echo', got %q", result.Binary)
	}
}

func TestParse_MultiLevelSubcommand(t *testing.T) {
	tests := []struct {
		input      string
		binary     string
		hasSubcmd  bool
	}{
		{"docker compose up -d", "docker", true},
		{"git remote add origin url", "git", true},
		{"kubectl get pods -n production", "kubectl", true},
		{"aws s3 cp file.txt s3://bucket/", "aws", true},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := Parse(tt.input)
			if result.Binary != tt.binary {
				t.Errorf("expected binary %q, got %q", tt.binary, result.Binary)
			}
			if tt.hasSubcmd && result.Subcommand == "" {
				t.Errorf("expected non-empty subcommand for %q", tt.input)
			}
		})
	}
}

func TestParse_CategoryDetection(t *testing.T) {
	tests := []struct {
		binary   string
		category string
	}{
		{"git", "git"},
		{"docker", "docker"},
		{"kubectl", "kubernetes"},
		{"ssh", "network"},
		{"curl", "network"},
		{"find", "filesystem"},
		{"tar", "archive"},
		{"grep", "text"},
		{"npm", "package"},
		{"ps", "process"},
		{"systemctl", "system"},
		{"make", "build"},
		{"python3", "language"},
		{"terraform", "infrastructure"},
		{"aws", "cloud"},
		{"unknown_tool", "other"},
	}

	for _, tt := range tests {
		t.Run(tt.binary, func(t *testing.T) {
			cat := detectCategory(tt.binary)
			if cat != tt.category {
				t.Errorf("expected category %q for %q, got %q", tt.category, tt.binary, cat)
			}
		})
	}
}

func TestParse_MaxLength(t *testing.T) {
	longCmd := "echo " + string(make([]byte, maxCommandLen+100))
	result := Parse(longCmd)
	if len(result.Raw) > maxCommandLen {
		t.Errorf("raw should be truncated to %d, got %d", maxCommandLen, len(result.Raw))
	}
}

func TestParse_CommandPrefixes(t *testing.T) {
	prefixes := []string{"sudo", "time", "nice", "nohup", "env", "command"}
	for _, prefix := range prefixes {
		t.Run(prefix, func(t *testing.T) {
			result := Parse(prefix + " ls -la")
			if result.Binary != "ls" {
				t.Errorf("expected prefix %q stripped, binary 'ls', got %q", prefix, result.Binary)
			}
		})
	}
}

func TestTokenize_BasicSplit(t *testing.T) {
	tokens := tokenize("git commit -m 'initial commit'")
	if len(tokens) < 3 {
		t.Errorf("expected at least 3 tokens, got %d: %v", len(tokens), tokens)
	}
	if tokens[0] != "git" {
		t.Errorf("expected first token 'git', got %q", tokens[0])
	}
}

func TestFirstSegment_NoPipe(t *testing.T) {
	result := firstSegment("git status")
	if result != "git status" {
		t.Errorf("expected 'git status', got %q", result)
	}
}

func TestFirstSegment_WithPipe(t *testing.T) {
	result := firstSegment("cat file | grep err")
	if result != "cat file" {
		t.Errorf("expected 'cat file', got %q", result)
	}
}

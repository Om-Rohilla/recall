package tests

import (
	"testing"

	"github.com/Om-Rohilla/recall/internal/capture"
	"github.com/Om-Rohilla/recall/pkg/config"
)

func TestParseSimpleCommand(t *testing.T) {
	tests := []struct {
		input       string
		wantBinary  string
		wantSub     string
		wantFlags   int
		wantCategory string
	}{
		{"ls -la", "ls", "", 1, "filesystem"},
		{"git status", "git", "status", 0, "git"},
		{"git commit -m 'test'", "git", "commit", 1, "git"},
		{"docker compose up -d", "docker", "compose up", 1, "docker"},
		{"find . -type f -name '*.go'", "find", "", 2, "filesystem"},
		{"kubectl get pods -n staging", "kubectl", "get pods", 1, "kubernetes"},
		{"curl -s https://api.example.com", "curl", "", 1, "network"},
		{"npm install express", "npm", "install express", 0, "package"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := capture.Parse(tt.input)
			if result.Binary != tt.wantBinary {
				t.Errorf("Parse(%q).Binary = %q, want %q", tt.input, result.Binary, tt.wantBinary)
			}
			if result.Subcommand != tt.wantSub {
				t.Errorf("Parse(%q).Subcommand = %q, want %q", tt.input, result.Subcommand, tt.wantSub)
			}
			if len(result.Flags) != tt.wantFlags {
				t.Errorf("Parse(%q).Flags = %d flags %v, want %d", tt.input, len(result.Flags), result.Flags, tt.wantFlags)
			}
			if result.Category != tt.wantCategory {
				t.Errorf("Parse(%q).Category = %q, want %q", tt.input, result.Category, tt.wantCategory)
			}
		})
	}
}

func TestParsePipedCommand(t *testing.T) {
	result := capture.Parse("cat file.txt | grep error | wc -l")
	if result.Binary != "cat" {
		t.Errorf("expected binary='cat' for piped command, got %q", result.Binary)
	}
	if result.Raw != "cat file.txt | grep error | wc -l" {
		t.Errorf("expected full raw command preserved, got %q", result.Raw)
	}
}

func TestParseChainedCommand(t *testing.T) {
	result := capture.Parse("mkdir -p build && cd build && cmake ..")
	if result.Binary != "mkdir" {
		t.Errorf("expected binary='mkdir' for chained command, got %q", result.Binary)
	}
}

func TestParseSudoPrefix(t *testing.T) {
	result := capture.Parse("sudo apt install nginx")
	if result.Binary != "apt" {
		t.Errorf("expected binary='apt' (not sudo), got %q", result.Binary)
	}
}

func TestParseEnvPrefix(t *testing.T) {
	result := capture.Parse("NODE_ENV=production npm start")
	if result.Binary != "npm" {
		t.Errorf("expected binary='npm' (not env var), got %q", result.Binary)
	}
}

func TestParseEmpty(t *testing.T) {
	result := capture.Parse("")
	if result.Binary != "" {
		t.Errorf("expected empty binary for empty input, got %q", result.Binary)
	}
}

func TestFilterSecrets(t *testing.T) {
	cfg := config.DefaultConfig()

	tests := []struct {
		input   string
		allowed bool
		reason  string
	}{
		{"git status", true, ""},
		{"export PASSWORD=secret123", false, "secret pattern"},
		{"curl -u admin:pass123 https://api.com", false, "secret pattern"},
		{"aws configure set aws_secret_access_key AKIA...", false, "secret pattern"},
		{"export API_KEY=abc123", false, "secret pattern"},
		{"echo my_token=xyz", false, "secret pattern"},
		{"docker build -t myapp .", true, ""},
		{"GITHUB_TOKEN=abc123 gh repo create", false, "secret pattern"},
		{"ssh-keygen -t ed25519", true, ""},
		{"export CREDENTIALS=test", false, "secret pattern"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := capture.Filter(tt.input, cfg)
			if result.Allowed != tt.allowed {
				t.Errorf("Filter(%q).Allowed = %v, want %v (reason: %s)", tt.input, result.Allowed, tt.allowed, result.Reason)
			}
		})
	}
}

func TestFilterNoise(t *testing.T) {
	cfg := config.DefaultConfig()

	tests := []struct {
		input   string
		allowed bool
	}{
		{"ls", false},
		{"cd", false},
		{"pwd", false},
		{"clear", false},
		{"exit", false},
		{"history", false},
		{"ls -la /var/log", false},
		{"git status", true},
		{"docker ps", true},
		{"find . -name '*.go'", true},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := capture.Filter(tt.input, cfg)
			if result.Allowed != tt.allowed {
				t.Errorf("Filter(%q).Allowed = %v, want %v", tt.input, result.Allowed, tt.allowed)
			}
		})
	}
}

func TestFilterEmpty(t *testing.T) {
	cfg := config.DefaultConfig()

	result := capture.Filter("", cfg)
	if result.Allowed {
		t.Error("expected empty command to be filtered")
	}

	result = capture.Filter("   ", cfg)
	if result.Allowed {
		t.Error("expected whitespace-only command to be filtered")
	}
}

func TestProcessHistoryLine(t *testing.T) {
	cfg := config.DefaultConfig()

	tests := []struct {
		input    string
		wantNil  bool
		wantBin  string
	}{
		{"git status", false, "git"},
		{": 1711382400:0;git commit -m 'test'", false, "git"},
		{"ls", true, ""},
		{"export PASSWORD=secret", true, ""},
		{"", true, ""},
		{"   ", true, ""},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := capture.ProcessHistoryLine(tt.input, cfg)
			if tt.wantNil && result != nil {
				t.Errorf("ProcessHistoryLine(%q) = %v, want nil", tt.input, result)
			}
			if !tt.wantNil {
				if result == nil {
					t.Fatalf("ProcessHistoryLine(%q) = nil, want non-nil", tt.input)
				}
				if result.Binary != tt.wantBin {
					t.Errorf("ProcessHistoryLine(%q).Binary = %q, want %q", tt.input, result.Binary, tt.wantBin)
				}
			}
		})
	}
}

func TestParseQuotedArgs(t *testing.T) {
	result := capture.Parse(`grep -rn "hello world" --include="*.go" .`)
	if result.Binary != "grep" {
		t.Errorf("expected binary='grep', got %q", result.Binary)
	}
	if len(result.Flags) < 2 {
		t.Errorf("expected at least 2 flags, got %d: %v", len(result.Flags), result.Flags)
	}
}

func TestCategoryDetection(t *testing.T) {
	tests := []struct {
		binary   string
		category string
	}{
		{"git", "git"},
		{"docker", "docker"},
		{"kubectl", "kubernetes"},
		{"find", "filesystem"},
		{"curl", "network"},
		{"npm", "package"},
		{"python", "language"},
		{"unknown-tool", "other"},
	}

	for _, tt := range tests {
		t.Run(tt.binary, func(t *testing.T) {
			result := capture.Parse(tt.binary + " --help")
			if result.Category != tt.category {
				t.Errorf("Parse(%q).Category = %q, want %q", tt.binary, result.Category, tt.category)
			}
		})
	}
}

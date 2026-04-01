package capture

import (
	"testing"

	"github.com/Om-Rohilla/recall/pkg/config"
)

func testConfig() *config.Config {
	cfg := config.DefaultConfig()
	cfg.Capture.Enabled = true
	cfg.Capture.NoiseFilter = true
	return cfg
}

func TestFilter_EmptyCommand(t *testing.T) {
	cfg := testConfig()
	result := Filter("", cfg)
	if result.Allowed {
		t.Error("empty command should be filtered")
	}
	if result.Reason != "empty command" {
		t.Errorf("expected reason 'empty command', got %q", result.Reason)
	}
}

func TestFilter_WhitespaceOnly(t *testing.T) {
	cfg := testConfig()
	result := Filter("   \t  ", cfg)
	if result.Allowed {
		t.Error("whitespace-only command should be filtered")
	}
}

func TestFilter_NormalCommand(t *testing.T) {
	cfg := testConfig()
	result := Filter("git status", cfg)
	if !result.Allowed {
		t.Errorf("normal command should be allowed, filtered with reason: %s", result.Reason)
	}
}

func TestFilter_NoiseCommands(t *testing.T) {
	cfg := testConfig()

	noiseCmds := []string{"ls", "cd", "pwd", "clear", "exit", "history"}
	for _, cmd := range noiseCmds {
		result := Filter(cmd, cfg)
		if result.Allowed {
			t.Errorf("noise command %q should be filtered", cmd)
		}
		if result.Reason != "noise command" {
			t.Errorf("noise command %q: expected reason 'noise command', got %q", cmd, result.Reason)
		}
	}
}

func TestFilter_NoiseDisabled(t *testing.T) {
	cfg := testConfig()
	cfg.Capture.NoiseFilter = false

	result := Filter("ls", cfg)
	if !result.Allowed {
		t.Error("noise filter disabled but command still filtered")
	}
}

func TestFilter_BuiltinSecretPatterns(t *testing.T) {
	cfg := testConfig()

	secretCmds := []struct {
		name string
		cmd  string
	}{
		{"password=", "curl -d password=mysecret http://example.com"},
		{"token=", "export TOKEN=abc123def"},
		{"api_key=", "curl -H 'api_key=secretkey' http://api.com"},
		{"bearer", "curl -H 'Authorization: Bearer abc.def.ghi' http://api.com"},
		{"ghp_ token", "git clone https://ghp_1234567890abcdef@github.com/repo"},
		{"sk- key", "openai api --api-key sk-1234567890abcdef"},
		{"BEGIN RSA", "echo '-----BEGIN RSA PRIVATE KEY-----'"},
		{"postgres://", "psql postgres://user:pass@host/db"},
		{"xoxb slack", "curl -H 'Authorization: Bearer xoxb-1234' https://slack.com/api"},
		{"pk_live_ stripe", "stripe listen --api-key pk_live_abc123"},
		{"eyJ jwt", "curl -H 'Authorization: eyJhbGciOiJIUzI1NiJ9.payload.sig' http://api.com"},
	}

	for _, tc := range secretCmds {
		t.Run(tc.name, func(t *testing.T) {
			result := Filter(tc.cmd, cfg)
			if result.Allowed {
				t.Errorf("secret pattern %q should be filtered for command: %s", tc.name, tc.cmd)
			}
			if result.Reason != "contains secret pattern" {
				t.Errorf("expected reason 'contains secret pattern', got %q", result.Reason)
			}
		})
	}
}

func TestFilter_CommandSpecificPatterns(t *testing.T) {
	cfg := testConfig()

	cmds := []struct {
		name string
		cmd  string
	}{
		{"curl -u", "curl -u user:pass http://example.com"},
		{"docker login -p", "docker login -p mysecretpass"},
		{"sshpass -p", "sshpass -p mypassword ssh user@host"},
		{"mysql -p", "mysql -p mydb"},
	}

	for _, tc := range cmds {
		t.Run(tc.name, func(t *testing.T) {
			result := Filter(tc.cmd, cfg)
			if result.Allowed {
				t.Errorf("command pattern %q should be filtered: %s", tc.name, tc.cmd)
			}
		})
	}
}

func TestFilter_RegexPatterns(t *testing.T) {
	cfg := testConfig()

	// Long hex string (looks like a token)
	result := Filter("curl -H 'X-Token: abcdef1234567890abcdef1234567890ab' http://api.com", cfg)
	if result.Allowed {
		t.Error("32+ char hex string should be filtered")
	}

	// Connection string with credentials
	result = Filter("mysql://root:superpass@localhost:3306/mydb", cfg)
	if result.Allowed {
		t.Error("connection string with credentials should be filtered")
	}
}

func TestFilter_UserPatterns(t *testing.T) {
	cfg := testConfig()
	cfg.Capture.SecretPatterns = append(cfg.Capture.SecretPatterns, "my_custom_secret")

	result := Filter("echo my_custom_secret=hello", cfg)
	if result.Allowed {
		t.Error("user-defined pattern should be filtered")
	}
}

func TestFilter_SafeCommands(t *testing.T) {
	cfg := testConfig()

	safeCmds := []string{
		"git commit -m 'fix bug'",
		"docker build -t myapp .",
		"kubectl get pods",
		"find / -name '*.go' -size +1M",
		"grep -rn 'TODO' .",
		"tar -czf backup.tar.gz ./src",
		"ssh user@host",
		"make build",
		"go test ./...",
		"npm install express",
	}

	for _, cmd := range safeCmds {
		t.Run(cmd, func(t *testing.T) {
			result := Filter(cmd, cfg)
			if !result.Allowed {
				t.Errorf("safe command should be allowed: %q (reason: %s)", cmd, result.Reason)
			}
		})
	}
}

func TestContainsSecret_CaseInsensitive(t *testing.T) {
	if !containsSecret("export PASSWORD=secret", nil) {
		t.Error("should detect PASSWORD= case-insensitively")
	}
	if !containsSecret("export password=secret", nil) {
		t.Error("should detect password= case-insensitively")
	}
}

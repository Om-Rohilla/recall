package tests

import (
	"testing"

	"github.com/Om-Rohilla/recall/internal/explain"
)

func TestExplainSimpleCommand(t *testing.T) {
	tests := []struct {
		input      string
		wantBinary string
		wantComps  int // minimum components
	}{
		{"ls -la", "ls", 2},
		{"git status", "git", 2},
		{"docker ps", "docker", 2},
		{"find . -name '*.log'", "find", 3},
		{"curl -s https://api.example.com", "curl", 3},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := explain.Explain(tt.input)
			if result.Binary != tt.wantBinary {
				t.Errorf("Explain(%q).Binary = %q, want %q", tt.input, result.Binary, tt.wantBinary)
			}
			if len(result.Components) < tt.wantComps {
				t.Errorf("Explain(%q) has %d components, want at least %d", tt.input, len(result.Components), tt.wantComps)
			}
		})
	}
}

func TestExplainFindCommand(t *testing.T) {
	result := explain.Explain("find . -type f -name '*.log' -mtime +30 -delete")

	if result.Binary != "find" {
		t.Errorf("expected binary='find', got %q", result.Binary)
	}

	if result.DangerLevel != explain.Destructive {
		t.Errorf("expected danger level Destructive for -delete, got %v", result.DangerLevel)
	}

	if len(result.Warnings) == 0 {
		t.Error("expected warnings for destructive -delete flag")
	}

	// Check that specific flags are explained
	flagsFound := map[string]bool{"-type": false, "-name": false, "-mtime": false, "-delete": false}
	for _, c := range result.Components {
		for flag := range flagsFound {
			if c.Token == flag || containsToken(c.Token, flag) {
				flagsFound[flag] = true
			}
		}
	}
	for flag, found := range flagsFound {
		if !found {
			t.Errorf("expected flag %q to be explained", flag)
		}
	}
}

func TestExplainTarCommand(t *testing.T) {
	result := explain.Explain("tar -xzvf archive.tar.gz -C /opt/")

	if result.Binary != "tar" {
		t.Errorf("expected binary='tar', got %q", result.Binary)
	}

	if len(result.Components) < 3 {
		t.Errorf("expected at least 3 components for tar command, got %d", len(result.Components))
	}
}

func TestExplainGitCommand(t *testing.T) {
	tests := []struct {
		input       string
		wantDanger  explain.DangerLevel
		wantSubcmd  bool
	}{
		{"git commit -m 'test'", explain.Safe, true},
		{"git push --force origin main", explain.Destructive, true},
		{"git reset --hard HEAD~1", explain.Destructive, true},
		{"git log --oneline --graph", explain.Safe, true},
		{"git branch -D feature/old", explain.Destructive, true},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := explain.Explain(tt.input)
			if result.Binary != "git" {
				t.Errorf("Explain(%q).Binary = %q, want 'git'", tt.input, result.Binary)
			}
			if result.DangerLevel != tt.wantDanger {
				t.Errorf("Explain(%q).DangerLevel = %v, want %v", tt.input, result.DangerLevel, tt.wantDanger)
			}
			if tt.wantSubcmd {
				hasSubcmd := false
				for _, c := range result.Components {
					if c.Type == "subcommand" {
						hasSubcmd = true
						break
					}
				}
				if !hasSubcmd {
					t.Errorf("Explain(%q) should have a subcommand component", tt.input)
				}
			}
		})
	}
}

func TestExplainDockerCommand(t *testing.T) {
	tests := []struct {
		input      string
		wantDanger explain.DangerLevel
	}{
		{"docker ps -a", explain.Safe},
		{"docker rm -f container1", explain.Destructive},
		{"docker run --privileged nginx", explain.Destructive},
		{"docker compose up -d", explain.Safe},
		{"docker system prune -af --volumes", explain.Destructive},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := explain.Explain(tt.input)
			if result.Binary != "docker" {
				t.Errorf("Explain(%q).Binary = %q, want 'docker'", tt.input, result.Binary)
			}
			if result.DangerLevel != tt.wantDanger {
				t.Errorf("Explain(%q).DangerLevel = %v, want %v", tt.input, result.DangerLevel, tt.wantDanger)
			}
		})
	}
}

func TestExplainRmCommand(t *testing.T) {
	result := explain.Explain("rm -rf /tmp/build/")

	if result.DangerLevel != explain.Destructive {
		t.Errorf("rm -rf should be Destructive, got %v", result.DangerLevel)
	}

	if len(result.Warnings) == 0 {
		t.Error("rm -rf should produce warnings")
	}
}

func TestExplainPipeline(t *testing.T) {
	result := explain.Explain("cat file.txt | grep error | wc -l")

	if result.Binary != "cat" {
		t.Errorf("expected first binary='cat', got %q", result.Binary)
	}

	pipeCount := 0
	for _, c := range result.Components {
		if c.Type == "pipe" {
			pipeCount++
		}
	}
	if pipeCount != 2 {
		t.Errorf("expected 2 pipe operators, got %d", pipeCount)
	}
}

func TestExplainChainedCommands(t *testing.T) {
	result := explain.Explain("mkdir -p build && cd build && cmake ..")

	operatorCount := 0
	for _, c := range result.Components {
		if c.Type == "operator" {
			operatorCount++
		}
	}
	if operatorCount != 2 {
		t.Errorf("expected 2 operators (&&), got %d", operatorCount)
	}
}

func TestExplainSudo(t *testing.T) {
	result := explain.Explain("sudo apt install nginx")

	// sudo should be a prefix, binary should be apt
	foundSudo := false
	foundApt := false
	for _, c := range result.Components {
		if c.Token == "sudo" {
			foundSudo = true
		}
		if c.Token == "apt" {
			foundApt = true
		}
	}
	if !foundSudo {
		t.Error("expected sudo as a component")
	}
	if !foundApt {
		t.Error("expected apt as a component")
	}
}

func TestExplainKubectl(t *testing.T) {
	result := explain.Explain("kubectl get pods -n staging -o wide")

	if result.Binary != "kubectl" {
		t.Errorf("expected binary='kubectl', got %q", result.Binary)
	}

	hasSubcmd := false
	for _, c := range result.Components {
		if c.Type == "subcommand" && c.Token == "get" {
			hasSubcmd = true
			break
		}
	}
	if !hasSubcmd {
		t.Error("expected 'get' as subcommand")
	}
}

func TestExplainRsync(t *testing.T) {
	result := explain.Explain("rsync -avz --delete ./src/ user@host:/deploy/")

	if result.DangerLevel != explain.Destructive {
		t.Errorf("rsync --delete should be Destructive, got %v", result.DangerLevel)
	}

	if len(result.Suggestions) == 0 {
		t.Error("rsync --delete should produce suggestions (--dry-run)")
	}
}

func TestExplainEmpty(t *testing.T) {
	result := explain.Explain("")
	if result.Binary != "" {
		t.Errorf("empty input should have empty binary, got %q", result.Binary)
	}
	if len(result.Components) != 0 {
		t.Errorf("empty input should have no components, got %d", len(result.Components))
	}
}

func TestDangerDetection(t *testing.T) {
	tests := []struct {
		input      string
		wantDanger explain.DangerLevel
	}{
		{"ls -la", explain.Safe},
		{"cat file.txt", explain.Safe},
		{"find . -delete", explain.Destructive},
		{"rm -rf /tmp/test", explain.Destructive},
		{"git push --force", explain.Destructive},
		{"git reset --hard HEAD", explain.Destructive},
		{"docker rm -f container", explain.Destructive},
		{"chmod -R 777 /var/www", explain.Caution},
		{"rsync --delete src/ dst/", explain.Destructive},
		{"sed -i 's/foo/bar/g' file", explain.Caution},
		{"kill -9 1234", explain.Destructive},
		{"iptables -F", explain.Destructive},
		{"crontab -r", explain.Destructive},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := explain.Explain(tt.input)
			if result.DangerLevel != tt.wantDanger {
				t.Errorf("Explain(%q).DangerLevel = %v, want %v", tt.input, result.DangerLevel, tt.wantDanger)
			}
		})
	}
}

func TestFlagLookup(t *testing.T) {
	tests := []struct {
		binary string
		flag   string
		wantOK bool
	}{
		{"find", "-name", true},
		{"find", "-delete", true},
		{"tar", "-x", true},
		{"git", "--force", true},
		{"docker", "-d", true},
		{"kubectl", "-n", true},
		{"ssh", "-L", true},
		{"grep", "-r", true},
		{"curl", "-X", true},
		{"rm", "-rf", true},
		{"unknown", "-x", false},
	}

	for _, tt := range tests {
		t.Run(tt.binary+"/"+tt.flag, func(t *testing.T) {
			info := explain.GetFlagInfo(tt.binary, tt.flag)
			if tt.wantOK && info == nil {
				t.Errorf("GetFlagInfo(%q, %q) = nil, want non-nil", tt.binary, tt.flag)
			}
			if !tt.wantOK && info != nil {
				t.Errorf("GetFlagInfo(%q, %q) = non-nil, want nil", tt.binary, tt.flag)
			}
		})
	}
}

func TestToolCount(t *testing.T) {
	count := explain.ToolCount()
	if count < 40 {
		t.Errorf("expected at least 40 tools in database, got %d", count)
	}
}

func TestSubcommandLookup(t *testing.T) {
	tests := []struct {
		binary     string
		subcommand string
		wantDesc   bool
	}{
		{"git", "commit", true},
		{"git", "push", true},
		{"docker", "run", true},
		{"docker", "compose", true},
		{"kubectl", "get", true},
		{"kubectl", "apply", true},
		{"systemctl", "start", true},
		{"npm", "install", true},
		{"go", "build", true},
		{"terraform", "apply", true},
		{"helm", "install", true},
		{"ls", "nonexistent", false},
	}

	for _, tt := range tests {
		t.Run(tt.binary+"/"+tt.subcommand, func(t *testing.T) {
			desc := explain.GetSubcommandDescription(tt.binary, tt.subcommand)
			if tt.wantDesc && desc == "" {
				t.Errorf("GetSubcommandDescription(%q, %q) returned empty, want description", tt.binary, tt.subcommand)
			}
			if !tt.wantDesc && desc != "" {
				t.Errorf("GetSubcommandDescription(%q, %q) = %q, want empty", tt.binary, tt.subcommand, desc)
			}
		})
	}
}

func TestExplainSSHWithTunnel(t *testing.T) {
	result := explain.Explain("ssh -L 8080:localhost:80 -N user@server.com")

	if result.Binary != "ssh" {
		t.Errorf("expected binary='ssh', got %q", result.Binary)
	}

	foundTunnel := false
	for _, c := range result.Components {
		if c.Token == "-L" {
			foundTunnel = true
			if c.Description == "" {
				t.Error("SSH -L flag should have a description")
			}
			break
		}
	}
	if !foundTunnel {
		t.Error("expected -L flag to be explained")
	}
}

func TestExplainCurlCommand(t *testing.T) {
	result := explain.Explain("curl -s -X POST -H 'Content-Type: application/json' -d '{\"key\":\"value\"}' https://api.example.com")

	if result.Binary != "curl" {
		t.Errorf("expected binary='curl', got %q", result.Binary)
	}

	// Should have multiple components
	if len(result.Components) < 5 {
		t.Errorf("expected at least 5 components for curl command, got %d", len(result.Components))
	}
}

func TestExplainEnvPrefix(t *testing.T) {
	result := explain.Explain("NODE_ENV=production npm start")

	foundEnv := false
	for _, c := range result.Components {
		if c.Token == "NODE_ENV=production" {
			foundEnv = true
			break
		}
	}
	if !foundEnv {
		t.Error("expected NODE_ENV=production as environment variable component")
	}
}

func TestWarningsContent(t *testing.T) {
	result := explain.Explain("find . -name '*.tmp' -delete")

	if len(result.Warnings) == 0 {
		t.Fatal("expected at least one warning for -delete")
	}

	hasDestructive := false
	for _, w := range result.Warnings {
		if w.Level == explain.Destructive {
			hasDestructive = true
			break
		}
	}
	if !hasDestructive {
		t.Error("expected a destructive warning for -delete")
	}
}

func TestSuggestions(t *testing.T) {
	result := explain.Explain("find . -name '*.log' -delete")

	if len(result.Suggestions) == 0 {
		t.Error("expected suggestions for find -delete command")
	}

	found := false
	for _, s := range result.Suggestions {
		if len(s) > 0 {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected non-empty suggestion text")
	}
}

func TestExplainCombinedFlags(t *testing.T) {
	result := explain.Explain("tar -czf archive.tar.gz .")

	foundCombined := false
	for _, c := range result.Components {
		if c.Token == "-czf" {
			foundCombined = true
			if c.Description == "" {
				t.Error("combined flag -czf should have description")
			}
			break
		}
	}
	if !foundCombined {
		t.Error("expected combined flag -czf to be explained")
	}
}

func TestExplainComponentTypes(t *testing.T) {
	result := explain.Explain("sudo find . -type f -name '*.log' | grep error")

	types := make(map[string]bool)
	for _, c := range result.Components {
		types[c.Type] = true
	}

	expectedTypes := []string{"binary", "flag", "pipe"}
	for _, et := range expectedTypes {
		if !types[et] {
			t.Errorf("expected component type %q in result", et)
		}
	}
}

func containsToken(component, token string) bool {
	return component == token || len(component) > len(token) && component[:len(token)] == token
}

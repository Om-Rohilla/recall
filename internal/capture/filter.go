package capture

import (
	"strings"

	"github.com/Om-Rohilla/recall/pkg/config"
)

// FilterResult describes why a command was filtered.
type FilterResult struct {
	Allowed bool
	Reason  string
}

// Filter checks whether a command should be stored in the vault.
func Filter(raw string, cfg *config.Config) FilterResult {
	raw = strings.TrimSpace(raw)

	if raw == "" {
		return FilterResult{Allowed: false, Reason: "empty command"}
	}

	if containsSecret(raw, cfg.Capture.SecretPatterns) {
		return FilterResult{Allowed: false, Reason: "contains secret pattern"}
	}

	if cfg.Capture.NoiseFilter && isNoise(raw, cfg.Capture.ExcludeCommands) {
		return FilterResult{Allowed: false, Reason: "noise command"}
	}

	return FilterResult{Allowed: true}
}

// containsSecret checks if the command contains any secret patterns.
func containsSecret(raw string, patterns []string) bool {
	lower := strings.ToLower(raw)
	for _, pattern := range patterns {
		if strings.Contains(lower, strings.ToLower(pattern)) {
			return true
		}
	}

	// Also catch inline secret assignments: export SECRET=..., KEY=value cmd
	if matchesSecretAssignment(lower) {
		return true
	}

	// Catch curl with auth: curl -u user:pass
	if strings.Contains(lower, "curl") && strings.Contains(lower, " -u ") {
		return true
	}

	return false
}

func matchesSecretAssignment(lower string) bool {
	secretEnvPatterns := []string{
		"password=", "passwd=", "token=", "secret=",
		"api_key=", "apikey=", "aws_secret", "private_key=",
		"credentials=", "auth_token=", "access_key=",
		"secret_key=", "client_secret=", "encryption_key=",
		"bearer ", "authorization:", "ghp_", "gho_", "github_token",
		"sk-", "sk_live_", "sk_test_",
		"-----begin", "-----begin rsa", "-----begin openssh",
		"mysql -p", "psql://", "mongodb+srv://",
		"://:@", "://user:pass@",
	}
	for _, p := range secretEnvPatterns {
		if strings.Contains(lower, p) {
			return true
		}
	}
	return false
}

// isNoise checks if a command is a noise command (too common/simple to store).
func isNoise(raw string, excludeList []string) bool {
	// Extract just the binary name from the raw command
	parts := strings.Fields(raw)
	if len(parts) == 0 {
		return true
	}

	binary := parts[0]

	// Strip leading path
	if idx := strings.LastIndex(binary, "/"); idx >= 0 {
		binary = binary[idx+1:]
	}

	for _, excluded := range excludeList {
		if binary == excluded {
			// Only filter bare commands — "cd" is noise but "cd /some/important/path" might not be
			// For Phase 1, filter all instances of excluded commands
			return true
		}
	}

	return false
}

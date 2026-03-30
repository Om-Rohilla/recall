package capture

import (
	"regexp"
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

// builtinSecretPatterns are always-on patterns that cannot be disabled.
// These catch common credential formats regardless of user config.
var builtinSecretPatterns = []string{
	// Generic credential patterns
	"password=", "passwd=", "token=", "secret=",
	"api_key=", "apikey=", "aws_secret", "private_key=",
	"credentials=", "auth_token=", "access_key=",
	"secret_key=", "client_secret=", "encryption_key=",
	// HTTP auth headers
	"bearer ", "authorization:", "authorization=",
	// GitHub tokens
	"ghp_", "gho_", "ghs_", "ghu_", "github_token", "github_pat_",
	// OpenAI / vendor-specific prefixes
	"sk-", "sk_live_", "sk_test_",
	// SSH/TLS key material
	"-----begin", "-----begin rsa", "-----begin openssh",
	"-----begin ec private", "-----begin private",
	// Database connection strings with credentials
	"psql://", "postgres://", "postgresql://",
	"mongodb+srv://", "mongodb://", "redis://:", "mysql://",
	"://:@", "://user:pass@",
	// Slack
	"xoxb-", "xoxp-", "xoxs-", "xoxa-",
	// Cloud provider secrets
	"azure_client_secret", "azure_tenant", "heroku_api_key",
	"slack_token", "slack_webhook_url", "sendgrid_api_key",
	"database_url=", "aws_session_token",
	"gcp_service_account", "google_application_credentials",
	// CI/CD tokens
	"travis_token", "circle_token", "circleci_token",
	"npm_token=", "npm_auth_token",
	"docker_password", "docker_auth",
	// Stripe
	"pk_live_", "pk_test_", "rk_live_", "rk_test_",
	// Twilio
	"twilio_auth_token",
	// JWT pattern (header.payload.signature)
	"eyj",
}

// builtinSecretRegexes catch patterns that require regex matching.
var builtinSecretRegexes = []*regexp.Regexp{
	// Environment variable export with inline value (export KEY=value or KEY=value cmd)
	regexp.MustCompile(`(?i)(export\s+)?[a-z_]*(?:secret|token|password|key|credential|auth)[a-z_]*\s*=\s*\S+`),
	// Long hex strings that look like tokens (32+ chars)
	regexp.MustCompile(`[0-9a-f]{32,}`),
	// Connection strings with embedded passwords (user:pass@host)
	regexp.MustCompile(`://[^/\s]+:[^/\s]+@[^/\s]+`),
}

// builtinCommandPatterns catch dangerous auth-passing CLI patterns.
var builtinCommandPatterns = []struct {
	binary string
	flags  []string
}{
	{"curl", []string{" -u ", " --user ", " --user="}},
	{"wget", []string{" --password", " --password=", " --http-password"}},
	{"docker", []string{" login -p", " login --password"}},
	{"sshpass", []string{" -p "}},
	{"mysql", []string{" -p"}},
	{"htpasswd", []string{" -b "}},
	{"vault", []string{" login ", " write "}},
	{"npm", []string{" set //:"}},
}

func containsSecret(raw string, userPatterns []string) bool {
	lower := strings.ToLower(raw)

	for _, pattern := range userPatterns {
		if strings.Contains(lower, strings.ToLower(pattern)) {
			return true
		}
	}

	for _, p := range builtinSecretPatterns {
		if strings.Contains(lower, p) {
			return true
		}
	}

	for _, cp := range builtinCommandPatterns {
		if strings.Contains(lower, cp.binary) {
			for _, flag := range cp.flags {
				if strings.Contains(lower, flag) {
					return true
				}
			}
		}
	}

	// Check regex patterns for more complex credential formats
	for _, re := range builtinSecretRegexes {
		if re.MatchString(raw) {
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

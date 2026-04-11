package capture

import (
	"math"
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

	// Phase 3: AST-based malicious injection guard.
	// Reject commands containing obfuscated subshells, process substitutions,
	// or dynamic evaluators (eval/source) that could poison the vault.
	astResult := CheckCommandAST(raw)
	if astResult.Trust == TrustLow {
		return FilterResult{Allowed: false, Reason: "low-trust command (AST): " + astResult.Reason}
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
	// Fly.io tokens
	"fo1_", "fo2_",
	// Sentry DSN (contains org slug — still sensitive)
	"sentry_dsn=", "sentry.dsn=",
	// Vercel (belt+suspenders alongside regex)
	"vercel_token=",
	// 1Password CLI output patterns
	"op://",
	// Doppler service/personal/staging tokens
	"dp.ct.", "dp.pt.", "dp.st.",
	// HashiCorp Vault tokens
	"hvs.", "hvb.", "hvr.",
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
			return true
		}
	}
	return false
}

// SanitizeSecrets scans the command for high-entropy tokens and signature
// matches, redacting them inline to preserve command structure.
func SanitizeSecrets(raw string, cfg *config.Config) string {
	// 1. Regex replacements (e.g. export AWS_SECRET=XXX)
	for _, re := range builtinSecretRegexes {
		raw = re.ReplaceAllStringFunc(raw, func(match string) string {
			idx := strings.Index(match, "=")
			if idx > 0 {
				return match[:idx+1] + "[REDACTED_SECRET]"
			}
			idx = strings.Index(match, ":")
			if idx > 0 {
				return match[:idx+1] + "[REDACTED_SECRET]"
			}
			return "[REDACTED_SECRET]"
		})
	}

	// 2. High Entropy Token Replacement
	// Matches typical base64, hex, or token alphabets > 16 chars
	tokenRe := regexp.MustCompile(`[a-zA-Z0-9_\-\.\+]{16,}`)
	raw = tokenRe.ReplaceAllStringFunc(raw, func(match string) string {
		if shannonEntropy(match) > 4.5 {
			return "[REDACTED_SECRET]"
		}
		return match
	})

	// 3. User & Builtin Secret Patterns (Prefix stripping)
	// (Note: this is brute force, but effective for things missed by entropy, like short tokens)
	for _, prefix := range builtinSecretPatterns {
		idx := strings.Index(strings.ToLower(raw), prefix)
		if idx >= 0 && !strings.Contains(strings.ToLower(raw), "[redacted") {
			// If it matches a known bad prefix and wasn't caught by entropy, it's safer to redact everything after it
			// Or we fall back to filtering the whole command.
			// Let's just redact the immediate next word.
			after := raw[idx+len(prefix):]
			fields := strings.Fields(after)
			if len(fields) > 0 {
				raw = strings.Replace(raw, fields[0], "[REDACTED_SECRET]", 1)
			}
		}
	}

	return raw
}

func shannonEntropy(data string) float64 {
	if len(data) == 0 {
		return 0
	}
	counts := make(map[rune]int)
	for _, char := range data {
		counts[char]++
	}
	var entropy float64
	length := float64(len(data))
	for _, count := range counts {
		prob := float64(count) / length
		entropy -= prob * math.Log2(prob)
	}
	return entropy
}


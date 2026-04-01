package errors

import (
	"fmt"
	"strings"
)

// UserError is a user-friendly error with actionable suggestions.
type UserError struct {
	Message    string   // User-facing message (with emoji)
	Cause      error    // Original error (for logging)
	Hints      []string // Actionable suggestions for the user
	HelpCmd    string   // A command the user can run for help
}

func (e *UserError) Error() string {
	var b strings.Builder
	b.WriteString(e.Message)
	if len(e.Hints) > 0 {
		b.WriteString("\n")
		for _, hint := range e.Hints {
			b.WriteString("  💡 " + hint + "\n")
		}
	}
	if e.HelpCmd != "" {
		b.WriteString("  ▸  " + e.HelpCmd + "\n")
	}
	return b.String()
}

func (e *UserError) Unwrap() error {
	return e.Cause
}

// Vault returns a friendly error for vault-related issues.
func Vault(cause error) *UserError {
	msg := ""
	if cause != nil {
		msg = cause.Error()
	}

	switch {
	case strings.Contains(msg, "no such file") || strings.Contains(msg, "not exist"):
		return &UserError{
			Message: "🔍 Vault not found",
			Cause:   cause,
			Hints:   []string{"Your vault hasn't been created yet."},
			HelpCmd: "recall init",
		}
	case strings.Contains(msg, "permission denied"):
		return &UserError{
			Message: "🔒 Permission denied accessing vault",
			Cause:   cause,
			Hints:   []string{"Check file permissions on your vault database."},
		}
	case strings.Contains(msg, "locked") || strings.Contains(msg, "busy"):
		return &UserError{
			Message: "⏳ Vault is busy",
			Cause:   cause,
			Hints:   []string{"Another recall process may be running.", "Wait a moment and try again."},
		}
	case strings.Contains(msg, "decrypt") || strings.Contains(msg, "cipher"):
		return &UserError{
			Message: "🔐 Could not decrypt vault",
			Cause:   cause,
			Hints:   []string{"Check your RECALL_VAULT_KEY environment variable.", "The encryption key may have changed."},
		}
	default:
		return &UserError{
			Message: "😕 Something went wrong with the vault",
			Cause:   cause,
			Hints:   []string{fmt.Sprintf("Internal error: %v", cause)},
			HelpCmd: "recall init",
		}
	}
}

// Search returns a friendly error for search-related issues.
func Search(cause error) *UserError {
	msg := ""
	if cause != nil {
		msg = cause.Error()
	}

	switch {
	case strings.Contains(msg, "FTS") || strings.Contains(msg, "fts"):
		return &UserError{
			Message: "🔎 Search index error",
			Cause:   cause,
			Hints:   []string{"The search index may be corrupted.", "Try rebuilding: recall import-history"},
		}
	case strings.Contains(msg, "no such table"):
		return &UserError{
			Message: "🔍 Vault needs setup",
			Cause:   cause,
			Hints:   []string{"Your vault database is missing tables."},
			HelpCmd: "recall init",
		}
	default:
		return &UserError{
			Message: "😅 Search hit a snag",
			Cause:   cause,
			Hints:   []string{"Try simpler or different keywords."},
		}
	}
}

// Shell returns a friendly error for shell-related issues.
func Shell(cause error) *UserError {
	msg := ""
	if cause != nil {
		msg = cause.Error()
	}

	switch {
	case strings.Contains(msg, "SHELL") || strings.Contains(msg, "not set"):
		return &UserError{
			Message: "🐚 Could not detect your shell",
			Cause:   cause,
			Hints:   []string{"The $SHELL environment variable isn't set.", "Specify manually with --shell flag."},
			HelpCmd: "recall init --shell zsh",
		}
	case strings.Contains(msg, "unsupported"):
		return &UserError{
			Message: "🐚 Unsupported shell",
			Cause:   cause,
			Hints:   []string{"Recall supports: zsh, bash, fish."},
		}
	default:
		return &UserError{
			Message: "🐚 Shell configuration error",
			Cause:   cause,
			Hints:   []string{fmt.Sprintf("Error: %v", cause)},
		}
	}
}

// Config returns a friendly error for config-related issues.
func Config(cause error) *UserError {
	return &UserError{
		Message: "⚙️ Configuration error",
		Cause:   cause,
		Hints:   []string{fmt.Sprintf("Error: %v", cause), "Try resetting with: recall config reset"},
	}
}

// Wrap wraps a generic error with a friendly message.
func Wrap(message string, cause error) *UserError {
	return &UserError{
		Message: message,
		Cause:   cause,
	}
}

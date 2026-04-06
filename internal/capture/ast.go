// Package capture provides AST-based shell command analysis to detect
// and reject potentially malicious or obfuscated command structures.
package capture

import (
	"strings"

	"mvdan.cc/sh/v3/syntax"
)

// TrustLevel indicates how trustworthy a command is based on AST analysis.
type TrustLevel int

const (
	// TrustHigh means the command is straightforward and safe to index.
	TrustHigh TrustLevel = iota
	// TrustLow means the command contains obfuscated or dynamic constructs.
	TrustLow
)

// ASTCheckResult holds the result of AST-based command analysis.
type ASTCheckResult struct {
	Trust  TrustLevel
	Reason string
}

// dangerousBuiltins are shell builtins that can execute arbitrary code
// or load external files, making them vectors for alias hooking / injection.
var dangerousBuiltins = map[string]bool{
	"eval":   true,
	"source": true,
	".":      true, // POSIX source alias
	"exec":   true,
}

// CheckCommandAST parses the raw command string using a POSIX-compatible
// shell parser and walks the AST to detect:
//   - Command substitutions ($(...) or backticks) — typically obfuscated payloads
//   - Process substitutions (<(...) or >(...))
//   - eval / source builtins — dynamic code execution vectors
//
// Returns TrustLow with a reason if any dangerous construct is found,
// otherwise TrustHigh.
func CheckCommandAST(raw string) ASTCheckResult {
	// Use POSIX/Bash-compatible parser.
	f, err := syntax.NewParser(syntax.Variant(syntax.LangBash)).Parse(
		strings.NewReader(raw), "",
	)
	if err != nil {
		// If we can't parse it, treat as low-trust (malformed/obfuscated).
		return ASTCheckResult{
			Trust:  TrustLow,
			Reason: "shell parse error — potentially obfuscated command",
		}
	}

	var found *ASTCheckResult

	// syntax.Walk visits every node; returning false stops descent into children.
	syntax.Walk(f, func(node syntax.Node) bool {
		if found != nil {
			return false // already flagged, stop walking
		}

		switch n := node.(type) {
		case *syntax.CmdSubst:
			// $(...) or `...` — command substitution
			r := ASTCheckResult{
				Trust:  TrustLow,
				Reason: "command substitution detected ($(...) or backticks)",
			}
			found = &r
			return false

		case *syntax.ProcSubst:
			// <(...) or >(...) — process substitution
			r := ASTCheckResult{
				Trust:  TrustLow,
				Reason: "process substitution detected (<(...) or >(...))",
			}
			found = &r
			return false

		case *syntax.CallExpr:
			// Check for dangerous builtins (eval, source, .)
			if len(n.Args) > 0 && len(n.Args[0].Parts) > 0 {
				if lit, ok := n.Args[0].Parts[0].(*syntax.Lit); ok {
					if dangerousBuiltins[lit.Value] {
						r := ASTCheckResult{
							Trust:  TrustLow,
							Reason: "dangerous builtin detected: " + lit.Value,
						}
						found = &r
						return false
					}
				}
			}
		}

		return true
	})

	if found != nil {
		return *found
	}
	return ASTCheckResult{Trust: TrustHigh}
}

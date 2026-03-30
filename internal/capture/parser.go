package capture

import (
	"strings"
	"unicode"
)

type ParsedCommand struct {
	Raw        string   `json:"raw"`
	Binary     string   `json:"binary"`
	Subcommand string   `json:"subcommand"`
	Flags      []string `json:"flags"`
	Args       []string `json:"args"`
	Category   string   `json:"category"`
}

const maxCommandLen = 10240 // 10 KB safety limit

// Parse breaks a raw command string into structured components.
// Handles pipes, &&, ;, and quoted arguments.
func Parse(raw string) ParsedCommand {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ParsedCommand{Raw: raw}
	}
	if len(raw) > maxCommandLen {
		raw = raw[:maxCommandLen]
	}

	// For piped/chained commands, parse only the first segment
	// but keep the full raw string
	segment := firstSegment(raw)
	tokens := tokenize(segment)

	if len(tokens) == 0 {
		return ParsedCommand{Raw: raw}
	}

	pc := ParsedCommand{
		Raw:    raw,
		Binary: tokens[0],
	}

	// Handle env var prefixes like "ENV=val command ..."
	for strings.Contains(pc.Binary, "=") && len(tokens) > 1 {
		tokens = tokens[1:]
		pc.Binary = tokens[0]
	}

	// Handle sudo, time, etc.
	for isCommandPrefix(pc.Binary) && len(tokens) > 1 {
		tokens = tokens[1:]
		pc.Binary = tokens[0]
	}

	if len(tokens) > 1 {
		rest := tokens[1:]
		pc.Subcommand, pc.Flags, pc.Args = classifyTokens(pc.Binary, rest)
	}

	pc.Category = detectCategory(pc.Binary)

	return pc
}

// firstSegment extracts the first command from a pipe/chain,
// respecting single and double quotes so that delimiters inside
// quoted strings are not treated as segment boundaries.
func firstSegment(raw string) string {
	inSingle := false
	inDouble := false
	escaped := false

	for i, r := range raw {
		if escaped {
			escaped = false
			continue
		}
		if r == '\\' && !inSingle {
			escaped = true
			continue
		}
		if r == '\'' && !inDouble {
			inSingle = !inSingle
			continue
		}
		if r == '"' && !inSingle {
			inDouble = !inDouble
			continue
		}
		if inSingle || inDouble {
			continue
		}
		if r == '|' || r == ';' {
			return strings.TrimSpace(raw[:i])
		}
		if r == '&' && i+1 < len(raw) && raw[i+1] == '&' {
			return strings.TrimSpace(raw[:i])
		}
	}
	return raw
}

// tokenize splits a command string respecting quoted segments.
func tokenize(s string) []string {
	var tokens []string
	var current strings.Builder
	inSingle := false
	inDouble := false
	escaped := false

	for _, r := range s {
		if escaped {
			current.WriteRune(r)
			escaped = false
			continue
		}

		if r == '\\' && !inSingle {
			escaped = true
			current.WriteRune(r)
			continue
		}

		if r == '\'' && !inDouble {
			inSingle = !inSingle
			current.WriteRune(r)
			continue
		}

		if r == '"' && !inSingle {
			inDouble = !inDouble
			current.WriteRune(r)
			continue
		}

		if unicode.IsSpace(r) && !inSingle && !inDouble {
			if current.Len() > 0 {
				tokens = append(tokens, current.String())
				current.Reset()
			}
			continue
		}

		current.WriteRune(r)
	}

	if current.Len() > 0 {
		tokens = append(tokens, current.String())
	}

	return tokens
}

// classifyTokens separates tokens into subcommand, flags, and args.
func classifyTokens(binary string, tokens []string) (subcommand string, flags []string, args []string) {
	subParts := []string{}
	seenFlag := false

	for _, t := range tokens {
		if strings.HasPrefix(t, "-") {
			seenFlag = true
			flags = append(flags, t)
		} else if !seenFlag && isLikelySubcommand(binary, t) {
			subParts = append(subParts, t)
		} else {
			args = append(args, t)
		}
	}

	subcommand = strings.Join(subParts, " ")
	return
}

// isLikelySubcommand checks if a token looks like a subcommand for known multi-level CLIs.
var multiCmdTools = map[string]bool{
	"git": true, "docker": true, "kubectl": true, "npm": true,
	"yarn": true, "cargo": true, "go": true, "pip": true, "pip3": true,
	"apt": true, "brew": true, "systemctl": true, "journalctl": true,
	"helm": true, "terraform": true, "aws": true, "gcloud": true,
	"az": true, "heroku": true, "gh": true, "flyctl": true,
	"podman": true, "snap": true, "dnf": true, "pacman": true,
	"pnpm": true, "bun": true, "deno": true, "rustup": true,
	"conda": true, "composer": true, "gradle": true, "mvn": true,
	"dotnet": true, "flutter": true, "firebase": true, "vercel": true,
}

func isLikelySubcommand(binary, token string) bool {
	if strings.HasPrefix(token, "-") || strings.HasPrefix(token, "/") || strings.HasPrefix(token, ".") {
		return false
	}
	return multiCmdTools[binary]
}

var commandPrefixes = map[string]bool{
	"sudo": true, "time": true, "nice": true, "nohup": true,
	"strace": true, "ltrace": true, "env": true, "command": true,
	"builtin": true, "exec": true,
}

func isCommandPrefix(s string) bool {
	return commandPrefixes[s]
}

var categoryMap = map[string]string{
	"git":        "git",
	"docker":     "docker",
	"kubectl":    "kubernetes",
	"helm":       "kubernetes",
	"terraform":  "infrastructure",
	"aws":        "cloud",
	"gcloud":     "cloud",
	"az":         "cloud",
	"ssh":        "network",
	"scp":        "network",
	"rsync":      "network",
	"curl":       "network",
	"wget":       "network",
	"ping":       "network",
	"netstat":    "network",
	"ss":         "network",
	"ip":         "network",
	"iptables":   "network",
	"find":       "filesystem",
	"ls":         "filesystem",
	"cp":         "filesystem",
	"mv":         "filesystem",
	"rm":         "filesystem",
	"mkdir":      "filesystem",
	"chmod":      "filesystem",
	"chown":      "filesystem",
	"du":         "filesystem",
	"df":         "filesystem",
	"tar":        "archive",
	"zip":        "archive",
	"unzip":      "archive",
	"gzip":       "archive",
	"grep":       "text",
	"sed":        "text",
	"awk":        "text",
	"sort":       "text",
	"cut":        "text",
	"wc":         "text",
	"head":       "text",
	"tail":       "text",
	"cat":        "text",
	"npm":        "package",
	"yarn":       "package",
	"pip":        "package",
	"cargo":      "package",
	"go":         "package",
	"apt":        "package",
	"brew":       "package",
	"ps":         "process",
	"kill":       "process",
	"top":        "process",
	"htop":       "process",
	"systemctl":  "system",
	"journalctl": "system",
	"make":       "build",
	"gcc":        "build",
	"python":     "language",
	"python3":    "language",
	"node":       "language",
	"ruby":       "language",
}

func detectCategory(binary string) string {
	if cat, ok := categoryMap[binary]; ok {
		return cat
	}
	return "other"
}

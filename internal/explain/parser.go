package explain

import (
	"fmt"
	"strings"
	"unicode"
)

// Component represents a single annotated piece of a command.
type Component struct {
	Token       string      `json:"token"`
	Type        string      `json:"type"` // "binary", "subcommand", "flag", "argument", "pipe", "redirect", "operator"
	Description string      `json:"description"`
	Danger      DangerLevel `json:"danger"`
	Tip         string      `json:"tip,omitempty"`
}

// ExplainResult is the full breakdown of a command.
type ExplainResult struct {
	Raw         string      `json:"raw"`
	Binary      string      `json:"binary"`
	Summary     string      `json:"summary"`
	Components  []Component `json:"components"`
	Warnings    []Warning   `json:"warnings,omitempty"`
	Suggestions []string    `json:"suggestions,omitempty"`
	DangerLevel DangerLevel `json:"danger_level"`
}

// Warning represents a safety warning about the command.
type Warning struct {
	Level   DangerLevel `json:"level"`
	Message string      `json:"message"`
}

// Explain decomposes a raw command string into annotated components.
func Explain(raw string) ExplainResult {
	raw = strings.TrimSpace(raw)
	result := ExplainResult{
		Raw:         raw,
		DangerLevel: Safe,
	}

	if raw == "" {
		return result
	}

	segments := splitPipeline(raw)
	for _, seg := range segments {
		explainSegment(&result, seg.Command)
		if seg.Operator != "" {
			result.Components = append(result.Components, Component{
				Token:       seg.Operator,
				Type:        operatorType(seg.Operator),
				Description: operatorDescription(seg.Operator),
				Danger:      Safe,
			})
		}
	}

	result.DangerLevel = worstDanger(result.Components)
	result.Warnings = buildWarnings(result)
	result.Suggestions = buildSuggestions(result)

	return result
}

type pipelineSegment struct {
	Command  string
	Operator string // "|", "&&", "||", ";", ">", ">>", "<", "2>&1", etc.
}

func splitPipeline(raw string) []pipelineSegment {
	var segments []pipelineSegment
	var current strings.Builder
	inSingle := false
	inDouble := false
	escaped := false

	flush := func(op string) {
		cmd := strings.TrimSpace(current.String())
		if cmd != "" || len(segments) == 0 {
			segments = append(segments, pipelineSegment{Command: cmd, Operator: op})
		}
		current.Reset()
	}

	runes := []rune(raw)
	for i := 0; i < len(runes); i++ {
		r := runes[i]

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
		if inSingle || inDouble {
			current.WriteRune(r)
			continue
		}

		// Check multi-char operators first
		if r == '&' && i+1 < len(runes) && runes[i+1] == '&' {
			flush("&&")
			i++
			continue
		}
		if r == '|' && i+1 < len(runes) && runes[i+1] == '|' {
			flush("||")
			i++
			continue
		}
		if r == '|' {
			flush("|")
			continue
		}
		if r == ';' {
			flush(";")
			continue
		}

		current.WriteRune(r)
	}

	cmd := strings.TrimSpace(current.String())
	if cmd != "" || len(segments) == 0 {
		if len(segments) == 0 {
			segments = append(segments, pipelineSegment{Command: cmd})
		} else {
			segments = append(segments, pipelineSegment{Command: cmd, Operator: ""})
		}
	}

	return segments
}

func explainSegment(result *ExplainResult, segment string) {
	tokens := tokenize(segment)
	if len(tokens) == 0 {
		return
	}

	idx := 0

	// Skip env var prefixes
	for idx < len(tokens) && strings.Contains(tokens[idx], "=") && !strings.HasPrefix(tokens[idx], "-") {
		result.Components = append(result.Components, Component{
			Token:       tokens[idx],
			Type:        "argument",
			Description: "set environment variable",
			Danger:      Safe,
		})
		idx++
	}
	if idx >= len(tokens) {
		return
	}

	// Skip command prefixes (sudo, time, etc.)
	for idx < len(tokens) && isPrefix(tokens[idx]) {
		result.Components = append(result.Components, Component{
			Token:       tokens[idx],
			Type:        "binary",
			Description: prefixDescription(tokens[idx]),
			Danger:      Safe,
		})
		idx++
	}
	if idx >= len(tokens) {
		return
	}

	binary := tokens[idx]
	if result.Binary == "" {
		result.Binary = binary
	}

	toolInfo := GetToolInfo(binary)
	binaryDesc := "execute " + binary
	if toolInfo != nil {
		binaryDesc = toolInfo.Description
	}

	result.Components = append(result.Components, Component{
		Token:       binary,
		Type:        "binary",
		Description: binaryDesc,
		Danger:      Safe,
	})
	idx++

	// Look for subcommands
	if toolInfo != nil && toolInfo.Subcommands != nil && idx < len(tokens) {
		token := tokens[idx]
		if desc, ok := toolInfo.Subcommands[token]; ok {
			result.Components = append(result.Components, Component{
				Token:       token,
				Type:        "subcommand",
				Description: desc,
				Danger:      Safe,
			})
			idx++

			// Docker has nested subcommands like "compose up", "system prune"
			if idx < len(tokens) {
				combined := token + " " + tokens[idx]
				_ = combined
				if subDesc := GetSubcommandDescription(binary, tokens[idx]); subDesc != "" {
					result.Components = append(result.Components, Component{
						Token:       tokens[idx],
						Type:        "subcommand",
						Description: subDesc,
						Danger:      Safe,
					})
					idx++
				}
			}
		}
	}

	// Process remaining tokens as flags, arguments, or redirects
	for idx < len(tokens) {
		token := tokens[idx]

		// Handle redirections
		if isRedirect(token) {
			comp := Component{
				Token:       token,
				Type:        "redirect",
				Description: redirectDescription(token),
				Danger:      Safe,
			}
			result.Components = append(result.Components, comp)
			// Next token is the redirect target
			if idx+1 < len(tokens) {
				idx++
				result.Components = append(result.Components, Component{
					Token:       tokens[idx],
					Type:        "argument",
					Description: "redirect target",
					Danger:      Safe,
				})
			}
			idx++
			continue
		}

		// Combined short flags like -czf, -xzvf, -la
		if isCombinedFlags(token) && toolInfo != nil {
			explainCombinedFlags(result, toolInfo, binary, token)
			idx++
			continue
		}

		// Handle flags
		if strings.HasPrefix(token, "-") {
			flagComp := explainFlag(toolInfo, binary, token)

			// Only apply generic value consumption for flags NOT recognized in the tool database.
			// Known flags are self-describing; unknown ones use the heuristic.
			if !isFlagRecognized(toolInfo, token) && flagTakesValue(token) && idx+1 < len(tokens) && !strings.HasPrefix(tokens[idx+1], "-") {
				idx++
				flagComp.Token = token + " " + tokens[idx]
				flagComp.Description = flagComp.Description + ": " + tokens[idx]
			}

			result.Components = append(result.Components, flagComp)
			idx++
			continue
		}

		// Regular argument
		desc := describeArgument(binary, token)
		result.Components = append(result.Components, Component{
			Token:       token,
			Type:        "argument",
			Description: desc,
			Danger:      Safe,
		})
		idx++
	}

	if result.Summary == "" {
		result.Summary = buildSummary(result)
	}
}

func explainFlag(toolInfo *ToolInfo, binary, flag string) Component {
	comp := Component{
		Token:  flag,
		Type:   "flag",
		Danger: Safe,
	}

	// Handle flags with = value (e.g., --output=file.txt)
	flagName := flag
	flagValue := ""
	if eqIdx := strings.Index(flag, "="); eqIdx > 0 {
		flagName = flag[:eqIdx]
		flagValue = flag[eqIdx+1:]
	}

	if toolInfo != nil {
		if info, ok := toolInfo.Flags[flagName]; ok {
			comp.Description = info.Description
			comp.Danger = info.Danger
			comp.Tip = info.Tip
			if flagValue != "" {
				comp.Description += ": " + flagValue
			}
			return comp
		}
		if info, ok := toolInfo.Flags[flag]; ok {
			comp.Description = info.Description
			comp.Danger = info.Danger
			comp.Tip = info.Tip
			return comp
		}
	}

	// Fallback: check global dangerous flag patterns
	danger, tip := checkGlobalDanger(binary, flag)
	comp.Danger = danger
	comp.Tip = tip
	comp.Description = "option " + flag

	return comp
}

func explainCombinedFlags(result *ExplainResult, toolInfo *ToolInfo, binary, combined string) {
	if toolInfo == nil {
		result.Components = append(result.Components, Component{
			Token:       combined,
			Type:        "flag",
			Description: "combined flags",
			Danger:      Safe,
		})
		return
	}

	// Check if the whole combined flag is known
	if info, ok := toolInfo.Flags[combined]; ok {
		result.Components = append(result.Components, Component{
			Token:       combined,
			Type:        "flag",
			Description: info.Description,
			Danger:      info.Danger,
			Tip:         info.Tip,
		})
		return
	}

	// Decompose combined flag into individual letters
	flags := combined[1:] // strip leading -
	var parts []string
	for _, ch := range flags {
		f := "-" + string(ch)
		if info, ok := toolInfo.Flags[f]; ok {
			parts = append(parts, info.Description)
		} else {
			parts = append(parts, string(ch))
		}
	}

	// Find worst danger in combined flags
	worst := Safe
	var tip string
	for _, ch := range flags {
		f := "-" + string(ch)
		if info, ok := toolInfo.Flags[f]; ok {
			if dangerRank(info.Danger) > dangerRank(worst) {
				worst = info.Danger
				tip = info.Tip
			}
		}
	}

	result.Components = append(result.Components, Component{
		Token:       combined,
		Type:        "flag",
		Description: strings.Join(parts, " + "),
		Danger:      worst,
		Tip:         tip,
	})
}

func isCombinedFlags(token string) bool {
	if !strings.HasPrefix(token, "-") || strings.HasPrefix(token, "--") {
		return false
	}
	inner := token[1:]
	if len(inner) <= 1 {
		return false
	}
	// All characters must be letters (no = or digits at start)
	for i, r := range inner {
		if i == 0 && !unicode.IsLetter(r) {
			return false
		}
		if !unicode.IsLetter(r) && !unicode.IsDigit(r) {
			return false
		}
	}
	return true
}

// checkGlobalDanger detects dangerous patterns regardless of tool.
var globalDangerFlags = map[string]struct {
	Danger DangerLevel
	Tip    string
}{
	"-rf":      {Destructive, "Recursive force delete — no recovery possible"},
	"-fr":      {Destructive, "Force recursive delete — no recovery possible"},
	"--force":  {Caution, "Forces the operation, bypassing safety checks"},
	"-f":       {Caution, "Force flag — bypasses confirmation prompts"},
	"--delete": {Destructive, "Deletes matching items"},
	"--purge":  {Destructive, "Removes items and their configuration"},
	"--hard":   {Destructive, "Discards changes permanently"},
}

func checkGlobalDanger(binary, flag string) (DangerLevel, string) {
	if info, ok := globalDangerFlags[flag]; ok {
		return info.Danger, info.Tip
	}
	return Safe, ""
}

func isFlagRecognized(toolInfo *ToolInfo, flag string) bool {
	if toolInfo == nil {
		return false
	}
	if _, ok := toolInfo.Flags[flag]; ok {
		return true
	}
	if eqIdx := strings.Index(flag, "="); eqIdx > 0 {
		_, ok := toolInfo.Flags[flag[:eqIdx]]
		return ok
	}
	return false
}

func flagTakesValue(flag string) bool {
	valueFlags := map[string]bool{
		"-f": true, "-o": true, "-n": true, "-C": true,
		"-m": true, "-p": true, "-u": true, "-e": true,
		"-i": true, "-d": true, "-t": true, "-k": true,
		"-H": true, "-X": true, "-w": true, "-b": true,
		"-c": true, "-I": true, "-P": true, "-s": true,
		"-name": true, "-iname": true, "-type": true,
		"-size": true, "-mtime": true, "-atime": true,
		"-ctime": true, "-mmin": true, "-maxdepth": true,
		"-mindepth": true, "-exec": true, "-path": true,
		"-perm": true, "-user": true, "-group": true,
		"-newer": true, "-regex": true,
		"--output": true, "--file": true, "--name": true,
		"--namespace": true, "--selector": true,
		"--context": true, "--kubeconfig": true,
		"--container": true, "--tail": true,
		"--since": true, "--set": true,
		"--values": true, "--timeout": true,
		"--replicas": true, "--exclude": true,
		"--include": true, "--bwlimit": true,
	}
	return valueFlags[flag]
}

func isPrefix(token string) bool {
	prefixes := map[string]bool{
		"sudo": true, "time": true, "nice": true, "nohup": true,
		"strace": true, "ltrace": true, "env": true, "command": true,
		"builtin": true, "exec": true, "watch": true,
	}
	return prefixes[token]
}

func prefixDescription(prefix string) string {
	descs := map[string]string{
		"sudo":    "run as superuser",
		"time":    "measure execution time",
		"nice":    "run with modified scheduling priority",
		"nohup":   "run immune to hangups",
		"strace":  "trace system calls",
		"ltrace":  "trace library calls",
		"env":     "run in modified environment",
		"command": "run command (bypass shell functions)",
		"builtin": "run shell builtin",
		"exec":    "replace shell with command",
		"watch":   "execute command periodically",
	}
	if desc, ok := descs[prefix]; ok {
		return desc
	}
	return prefix
}

func isRedirect(token string) bool {
	redirects := map[string]bool{
		">": true, ">>": true, "<": true, "2>": true,
		"2>>": true, "&>": true, "2>&1": true, "<<<": true,
	}
	return redirects[token]
}

func redirectDescription(token string) string {
	descs := map[string]string{
		">":    "redirect stdout (overwrite)",
		">>":   "redirect stdout (append)",
		"<":    "redirect stdin from file",
		"2>":   "redirect stderr (overwrite)",
		"2>>":  "redirect stderr (append)",
		"&>":   "redirect stdout and stderr",
		"2>&1": "redirect stderr to stdout",
		"<<<":  "here-string input",
	}
	if desc, ok := descs[token]; ok {
		return desc
	}
	return "redirect"
}

func operatorType(op string) string {
	switch op {
	case "|":
		return "pipe"
	case "&&", "||", ";":
		return "operator"
	default:
		return "operator"
	}
}

func operatorDescription(op string) string {
	descs := map[string]string{
		"|":  "pipe output to next command",
		"&&": "run next command only if this succeeds",
		"||": "run next command only if this fails",
		";":  "run next command regardless",
	}
	if desc, ok := descs[op]; ok {
		return desc
	}
	return op
}

func describeArgument(binary, arg string) string {
	// Describe common argument patterns
	if arg == "." {
		return "current directory"
	}
	if arg == ".." {
		return "parent directory"
	}
	if arg == "/" {
		return "root directory"
	}
	if arg == "-" {
		return "read from stdin / previous directory"
	}
	if arg == "--" {
		return "end of options"
	}
	if arg == "./..." {
		return "all packages recursively"
	}
	if strings.HasPrefix(arg, "~/") {
		return "path in home directory"
	}
	if strings.HasPrefix(arg, "/") {
		return "absolute path"
	}
	if strings.Contains(arg, "://") {
		return "URL"
	}
	if strings.Contains(arg, "@") && strings.Contains(arg, ":") {
		return "remote host (user@host:path)"
	}
	if strings.Contains(arg, "@") {
		return "remote target (user@host)"
	}
	if strings.HasPrefix(arg, "'") || strings.HasPrefix(arg, "\"") {
		inner := strings.Trim(arg, "'\"")
		if strings.Contains(inner, "*") || strings.Contains(inner, "?") {
			return fmt.Sprintf("pattern: %s", inner)
		}
		return fmt.Sprintf("literal: %s", inner)
	}
	if strings.Contains(arg, "*") || strings.Contains(arg, "?") {
		return "glob pattern"
	}
	if strings.Contains(arg, ".tar") || strings.Contains(arg, ".gz") || strings.Contains(arg, ".zip") || strings.Contains(arg, ".bz2") {
		return "archive file"
	}
	return "argument"
}

func buildSummary(result *ExplainResult) string {
	var parts []string
	for _, c := range result.Components {
		switch c.Type {
		case "binary":
			parts = append(parts, c.Description)
		case "subcommand":
			parts = append(parts, c.Description)
		}
	}
	if len(parts) == 0 {
		return result.Binary
	}
	return strings.Join(parts, " — ")
}

func buildWarnings(result ExplainResult) []Warning {
	var warnings []Warning
	seen := make(map[string]bool)

	for _, c := range result.Components {
		if c.Danger == Destructive {
			msg := fmt.Sprintf("DESTRUCTIVE: %s (%s)", c.Description, c.Token)
			if !seen[msg] {
				warnings = append(warnings, Warning{Level: Destructive, Message: msg})
				seen[msg] = true
			}
		} else if c.Danger == Caution {
			msg := fmt.Sprintf("CAUTION: %s (%s)", c.Description, c.Token)
			if !seen[msg] {
				warnings = append(warnings, Warning{Level: Caution, Message: msg})
				seen[msg] = true
			}
		}
	}

	// Pattern-based warnings
	raw := strings.ToLower(result.Raw)
	if strings.Contains(raw, "rm -rf /") || strings.Contains(raw, "rm -fr /") {
		w := "CRITICAL: This deletes everything on the filesystem. Do NOT run this."
		if !seen[w] {
			warnings = append(warnings, Warning{Level: Destructive, Message: w})
		}
	}
	if strings.Contains(raw, ":(){ :|:& };:") {
		w := "CRITICAL: This is a fork bomb — will crash the system."
		if !seen[w] {
			warnings = append(warnings, Warning{Level: Destructive, Message: w})
		}
	}
	if strings.Contains(raw, "mkfs") {
		w := "DESTRUCTIVE: mkfs formats a partition — all data will be lost."
		if !seen[w] {
			warnings = append(warnings, Warning{Level: Destructive, Message: w})
		}
	}
	if strings.Contains(raw, "dd") && strings.Contains(raw, "of=/dev/") {
		w := "DESTRUCTIVE: Writing directly to a device — wrong target will destroy data."
		if !seen[w] {
			warnings = append(warnings, Warning{Level: Destructive, Message: w})
		}
	}
	if strings.Contains(raw, "drop table") || strings.Contains(raw, "drop database") {
		w := "DESTRUCTIVE: SQL DROP permanently deletes the table/database."
		if !seen[w] {
			warnings = append(warnings, Warning{Level: Destructive, Message: w})
		}
	}
	if strings.Contains(raw, "chmod 777") || strings.Contains(raw, "chmod -r 777") {
		w := "CAUTION: chmod 777 makes files world-readable/writable — security risk."
		if !seen[w] {
			warnings = append(warnings, Warning{Level: Caution, Message: w})
		}
	}

	return warnings
}

func buildSuggestions(result ExplainResult) []string {
	var suggestions []string
	raw := strings.ToLower(result.Raw)

	for _, c := range result.Components {
		if c.Tip != "" {
			suggestions = append(suggestions, c.Tip)
		}
	}

	// Add contextual suggestions
	if strings.Contains(raw, "-delete") && strings.Contains(raw, "find") {
		suggestions = appendUnique(suggestions, "Remove -delete and add -print first to preview what would be deleted")
	}
	if strings.Contains(raw, "--force") && strings.Contains(raw, "push") {
		suggestions = appendUnique(suggestions, "Consider --force-with-lease instead to prevent overwriting others' changes")
	}
	if strings.Contains(raw, "git reset --hard") {
		suggestions = appendUnique(suggestions, "Use --soft to keep changes staged, or --mixed to unstage but keep in working tree")
	}
	if strings.Contains(raw, "rm -rf") {
		suggestions = appendUnique(suggestions, "Use rm -ri for interactive confirmation, or ls first to preview")
	}

	return suggestions
}

func appendUnique(slice []string, s string) []string {
	for _, existing := range slice {
		if existing == s {
			return slice
		}
	}
	return append(slice, s)
}

func worstDanger(components []Component) DangerLevel {
	worst := Safe
	for _, c := range components {
		if dangerRank(c.Danger) > dangerRank(worst) {
			worst = c.Danger
		}
	}
	return worst
}

func dangerRank(d DangerLevel) int {
	switch d {
	case Safe:
		return 0
	case Caution:
		return 1
	case Destructive:
		return 2
	default:
		return 0
	}
}

// tokenize splits a command string respecting quoted segments.
func tokenize(s string) []string {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}

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

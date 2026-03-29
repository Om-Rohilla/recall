package intelligence

import (
	"regexp"
	"strings"
)

var (
	pathPattern     = regexp.MustCompile(`(?:^|[\s=])(/[^\s]+|\.{1,2}/[^\s]+|~/[^\s]+)`)
	urlPattern      = regexp.MustCompile(`https?://[^\s]+`)
	ipPattern       = regexp.MustCompile(`\b\d{1,3}\.\d{1,3}\.\d{1,3}\.\d{1,3}\b`)
	portPattern     = regexp.MustCompile(`(?::)\d{2,5}\b`)
	hashPattern     = regexp.MustCompile(`\b[0-9a-f]{7,40}\b`)
	versionPattern  = regexp.MustCompile(`\b\d+\.\d+(?:\.\d+)?\b`)
	quotedPattern   = regexp.MustCompile(`(?:"[^"]*"|'[^']*')`)
	branchPattern   = regexp.MustCompile(`(?:feature|bugfix|hotfix|release|fix)/[^\s]+`)
	containerID     = regexp.MustCompile(`\b[0-9a-f]{12,64}\b`)
)

// ExtractPattern converts a concrete command into a template by replacing
// variable parts (paths, URLs, hashes, etc.) with placeholders.
func ExtractPattern(raw string) string {
	template := raw

	template = urlPattern.ReplaceAllString(template, "{url}")
	template = ipPattern.ReplaceAllString(template, "{ip}")
	template = branchPattern.ReplaceAllString(template, "{branch}")
	template = hashPattern.ReplaceAllString(template, "{hash}")
	template = containerID.ReplaceAllString(template, "{id}")
	template = portPattern.ReplaceAllString(template, ":{port}")
	template = versionPattern.ReplaceAllString(template, "{version}")

	// Replace quoted strings but preserve flag-like quoted args
	template = quotedPattern.ReplaceAllStringFunc(template, func(match string) string {
		inner := match[1 : len(match)-1]
		if strings.HasPrefix(inner, "-") {
			return match
		}
		return "{arg}"
	})

	// Replace absolute paths but preserve short well-known paths
	template = pathPattern.ReplaceAllStringFunc(template, func(match string) string {
		trimmed := strings.TrimSpace(match)
		// Keep common short paths as-is
		wellKnown := []string{"/dev/null", "/etc/", "/tmp/", "/var/", "/usr/", "./", "../"}
		for _, wk := range wellKnown {
			if strings.HasPrefix(trimmed, wk) && len(trimmed) <= len(wk)+5 {
				return match
			}
		}
		prefix := ""
		if match != trimmed {
			prefix = match[:len(match)-len(trimmed)]
		}
		return prefix + "{path}"
	})

	return template
}

// NormalizePattern cleans up a pattern template for deduplication.
func NormalizePattern(pattern string) string {
	pattern = strings.TrimSpace(pattern)
	fields := strings.Fields(pattern)
	return strings.Join(fields, " ")
}

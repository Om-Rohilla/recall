package intelligence

import (
	"strings"
	"unicode"
)

type Intent struct {
	RawQuery       string   `json:"raw_query"`
	Tokens         []string `json:"tokens"`
	ExpandedTokens []string `json:"expanded_tokens"`
	Category       string   `json:"category,omitempty"`
	Action         string   `json:"action,omitempty"`
	Target         string   `json:"target,omitempty"`
}

func ExtractIntent(query string) Intent {
	intent := Intent{RawQuery: query}

	intent.Tokens = tokenizeQuery(query)
	if len(intent.Tokens) == 0 {
		return intent
	}

	intent.Action = detectAction(intent.Tokens)
	intent.Target = detectTarget(intent.Tokens)
	intent.Category = detectQueryCategory(intent.Tokens)
	intent.ExpandedTokens = expandSynonyms(intent.Tokens)

	return intent
}

func tokenizeQuery(query string) []string {
	query = strings.ToLower(strings.TrimSpace(query))
	if query == "" {
		return nil
	}

	words := strings.FieldsFunc(query, func(r rune) bool {
		return unicode.IsSpace(r) || r == ',' || r == ';'
	})

	var tokens []string
	for _, w := range words {
		cleaned := strings.Map(func(r rune) rune {
			if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' || r == '_' || r == '.' || r == '/' || r == '+' {
				return r
			}
			return -1
		}, w)
		if cleaned != "" && !isStopWord(cleaned) {
			tokens = append(tokens, cleaned)
		}
	}
	return tokens
}

var stopWords = map[string]bool{
	"a": true, "an": true, "the": true, "to": true, "in": true,
	"on": true, "at": true, "for": true, "of": true, "with": true,
	"and": true, "or": true, "is": true, "it": true, "this": true,
	"that": true, "from": true, "by": true, "as": true, "be": true,
	"was": true, "are": true, "been": true, "being": true, "have": true,
	"has": true, "do": true, "does": true, "did": true, "will": true,
	"would": true, "could": true, "should": true, "may": true, "might": true,
	"can": true, "i": true, "me": true, "my": true, "how": true,
	"what": true, "which": true, "where": true, "when": true, "who": true,
}

func isStopWord(word string) bool {
	return stopWords[word]
}

func expandSynonyms(tokens []string) []string {
	seen := make(map[string]bool)
	var expanded []string

	for _, t := range tokens {
		if !seen[t] {
			seen[t] = true
			expanded = append(expanded, t)
		}
		if syns, ok := synonymDict[t]; ok {
			for _, s := range syns {
				if !seen[s] {
					seen[s] = true
					expanded = append(expanded, s)
				}
			}
		}
	}
	return expanded
}

var actionWords = map[string]string{
	"find":       "find",
	"search":     "find",
	"locate":     "find",
	"look":       "find",
	"show":       "show",
	"list":       "show",
	"display":    "show",
	"print":      "show",
	"view":       "show",
	"get":        "show",
	"check":      "show",
	"see":        "show",
	"delete":     "delete",
	"remove":     "delete",
	"rm":         "delete",
	"purge":      "delete",
	"clean":      "delete",
	"cleanup":    "delete",
	"clear":      "delete",
	"wipe":       "delete",
	"kill":       "stop",
	"stop":       "stop",
	"terminate":  "stop",
	"halt":       "stop",
	"end":        "stop",
	"abort":      "stop",
	"create":     "create",
	"make":       "create",
	"new":        "create",
	"add":        "create",
	"generate":   "create",
	"init":       "create",
	"initialize": "create",
	"setup":      "create",
	"copy":       "copy",
	"cp":         "copy",
	"duplicate":  "copy",
	"clone":      "copy",
	"move":       "move",
	"mv":         "move",
	"rename":     "move",
	"transfer":   "move",
	"change":     "modify",
	"edit":       "modify",
	"modify":     "modify",
	"update":     "modify",
	"set":        "modify",
	"configure":  "modify",
	"install":    "install",
	"uninstall":  "install",
	"upgrade":    "install",
	"downgrade":  "install",
	"run":        "run",
	"execute":    "run",
	"start":      "run",
	"launch":     "run",
	"open":       "run",
	"compress":   "archive",
	"extract":    "archive",
	"zip":        "archive",
	"unzip":      "archive",
	"tar":        "archive",
	"archive":    "archive",
	"pack":       "archive",
	"unpack":     "archive",
	"connect":    "connect",
	"ssh":        "connect",
	"login":      "connect",
	"mount":      "connect",
	"undo":       "undo",
	"revert":     "undo",
	"reset":      "undo",
	"rollback":   "undo",
	"restore":    "undo",
	"monitor":    "monitor",
	"watch":      "monitor",
	"log":        "monitor",
	"tail":       "monitor",
	"debug":      "monitor",
	"trace":      "monitor",
}

func detectAction(tokens []string) string {
	for _, t := range tokens {
		if action, ok := actionWords[t]; ok {
			return action
		}
	}
	return ""
}

var targetWords = map[string]string{
	"file":        "file",
	"files":       "file",
	"directory":   "directory",
	"folder":      "directory",
	"dir":         "directory",
	"directories": "directory",
	"folders":     "directory",
	"process":     "process",
	"processes":   "process",
	"pid":         "process",
	"port":        "port",
	"ports":       "port",
	"container":   "container",
	"containers":  "container",
	"image":       "image",
	"images":      "image",
	"volume":      "volume",
	"volumes":     "volume",
	"network":     "network",
	"branch":      "branch",
	"branches":    "branch",
	"commit":      "commit",
	"commits":     "commit",
	"tag":         "tag",
	"tags":        "tag",
	"remote":      "remote",
	"stash":       "stash",
	"pod":         "pod",
	"pods":        "pod",
	"service":     "service",
	"services":    "service",
	"deployment":  "deployment",
	"namespace":   "namespace",
	"package":     "package",
	"packages":    "package",
	"dependency":  "package",
	"dependencies": "package",
	"module":      "package",
	"modules":     "package",
	"user":        "user",
	"users":       "user",
	"group":       "group",
	"permission":  "permission",
	"permissions": "permission",
	"key":         "key",
	"keys":        "key",
	"disk":        "disk",
	"space":       "disk",
	"memory":      "memory",
	"ram":         "memory",
	"cpu":         "cpu",
	"log":         "log",
	"logs":        "log",
}

func detectTarget(tokens []string) string {
	for _, t := range tokens {
		if target, ok := targetWords[t]; ok {
			return target
		}
	}
	return ""
}

var categoryKeywords = map[string][]string{
	"git":            {"git", "commit", "branch", "merge", "rebase", "stash", "cherry-pick", "bisect", "blame", "diff", "pull", "push", "fetch", "clone", "checkout", "tag", "remote"},
	"docker":         {"docker", "container", "image", "compose", "dockerfile", "volume", "swarm", "registry"},
	"kubernetes":     {"kubectl", "kubernetes", "k8s", "pod", "deployment", "service", "namespace", "ingress", "helm", "kube"},
	"filesystem":     {"file", "files", "directory", "folder", "find", "search", "locate", "disk", "space", "size", "permission", "chmod", "chown", "symlink", "link"},
	"network":        {"ssh", "port", "curl", "wget", "http", "https", "dns", "ip", "firewall", "proxy", "tunnel", "socket", "ping", "traceroute", "netstat"},
	"process":        {"process", "kill", "pid", "signal", "zombie", "daemon", "background", "foreground", "jobs", "top", "htop"},
	"archive":        {"tar", "zip", "gzip", "compress", "extract", "archive", "unzip", "untar", "bzip", "xz", "7z"},
	"text":           {"grep", "sed", "awk", "sort", "cut", "wc", "head", "tail", "cat", "less", "more", "regex", "pattern", "replace", "filter"},
	"package":        {"npm", "yarn", "pip", "cargo", "apt", "brew", "install", "package", "dependency", "module"},
	"system":         {"systemctl", "service", "cron", "crontab", "startup", "boot", "shutdown", "reboot", "uptime"},
	"cloud":          {"aws", "gcloud", "azure", "s3", "ec2", "lambda", "cloudfront"},
	"infrastructure": {"terraform", "ansible", "vagrant", "packer"},
}

func detectQueryCategory(tokens []string) string {
	scores := make(map[string]int)
	for _, t := range tokens {
		for cat, keywords := range categoryKeywords {
			for _, kw := range keywords {
				if t == kw {
					scores[cat]++
				}
			}
		}
	}

	bestCat := ""
	bestScore := 0
	for cat, score := range scores {
		if score > bestScore {
			bestScore = score
			bestCat = cat
		}
	}
	return bestCat
}

// BuildFTSQuery creates an FTS5 query string from expanded intent tokens.
func BuildFTSQuery(intent Intent) string {
	if len(intent.ExpandedTokens) == 0 {
		return ""
	}

	var terms []string
	for _, t := range intent.ExpandedTokens {
		cleaned := strings.Map(func(r rune) rune {
			if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '_' || r == '-' || r == '.' || r == '/' {
				return r
			}
			return -1
		}, t)
		if cleaned != "" {
			terms = append(terms, cleaned)
		}
	}

	if len(terms) == 0 {
		return ""
	}

	return strings.Join(terms, " OR ")
}

var synonymDict = map[string][]string{
	"find":       {"search", "locate", "look"},
	"search":     {"find", "locate", "look"},
	"locate":     {"find", "search"},
	"delete":     {"remove", "rm", "erase", "purge"},
	"remove":     {"delete", "rm", "erase", "purge"},
	"kill":       {"terminate", "stop", "end", "signal"},
	"stop":       {"kill", "terminate", "halt", "end"},
	"big":        {"large", "huge", "heavy"},
	"large":      {"big", "huge", "heavy"},
	"small":      {"tiny", "little", "minimal"},
	"show":       {"list", "display", "print", "view"},
	"list":       {"show", "display", "print", "view"},
	"create":     {"make", "new", "generate", "init"},
	"copy":       {"cp", "duplicate", "clone"},
	"move":       {"mv", "rename", "transfer"},
	"compress":   {"zip", "tar", "gzip", "archive", "pack"},
	"extract":    {"unzip", "untar", "decompress", "unpack"},
	"install":    {"setup", "add"},
	"uninstall":  {"remove", "purge"},
	"update":     {"upgrade", "refresh"},
	"process":    {"pid", "task"},
	"port":       {"socket", "listen"},
	"file":       {"files"},
	"files":      {"file"},
	"folder":     {"directory", "dir"},
	"directory":  {"folder", "dir"},
	"container":  {"containers"},
	"image":      {"images"},
	"branch":     {"branches"},
	"commit":     {"commits"},
	"disk":       {"storage", "space"},
	"space":      {"disk", "storage", "usage"},
	"size":       {"space", "disk"},
	"permission": {"chmod", "access", "rights"},
	"undo":       {"revert", "reset", "rollback"},
	"revert":     {"undo", "reset", "rollback"},
	"log":        {"logs", "history", "journal"},
	"connect":    {"ssh", "login", "remote"},
	"ssh":        {"connect", "remote"},
	"download":   {"fetch", "get", "pull", "wget", "curl"},
	"upload":     {"push", "send", "deploy"},
	"run":        {"execute", "start", "launch"},
	"restart":    {"reload", "refresh"},
	"monitor":    {"watch", "observe", "track"},
	"debug":      {"trace", "inspect", "diagnose"},
	"test":       {"check", "verify", "validate"},
	"clean":      {"cleanup", "clear", "purge", "prune"},
	"cleanup":    {"clean", "clear", "purge", "prune"},
	"config":     {"configure", "configuration", "settings"},
	"setup":      {"configure", "init", "initialize"},
	"env":        {"environment", "variable", "variables"},
	"network":    {"net", "networking"},
	"memory":     {"ram", "mem"},
	"cpu":        {"processor", "cores"},
	"user":       {"account"},
	"password":   {"passwd", "credential"},
	"key":        {"keys", "ssh-key"},
	"git":        {},
	"docker":     {},
	"kubectl":    {"kube", "kubernetes"},
	"all":        {"every", "each"},
	"old":        {"aged", "stale", "outdated"},
	"new":        {"recent", "latest", "fresh"},
	"running":    {"active", "started"},
	"stopped":    {"inactive", "dead", "exited"},
	"failed":     {"error", "broken", "crashed"},
}

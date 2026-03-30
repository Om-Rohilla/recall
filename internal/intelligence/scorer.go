package intelligence

import (
	"math"
	"strings"
	"time"

	appctx "github.com/Om-Rohilla/recall/internal/context"
	"github.com/Om-Rohilla/recall/internal/vault"
)

const (
	WeightText      = 0.30
	WeightIntent    = 0.25
	WeightFrequency = 0.20
	WeightContext   = 0.15
	WeightRecency   = 0.10
)

type ScoredResult struct {
	Command        vault.Command `json:"command"`
	FinalScore     float64       `json:"final_score"`
	Confidence     float64       `json:"confidence"`
	TextScore      float64       `json:"text_score"`
	IntentScore    float64       `json:"intent_score"`
	FrequencyScore float64       `json:"frequency_score"`
	ContextScore   float64       `json:"context_score"`
	RecencyScore   float64       `json:"recency_score"`
	MatchType      string        `json:"match_type"`
}

type ScoringInput struct {
	Command      vault.Command
	Contexts     []vault.Context
	FTSRank      float64
	MaxFrequency int
	MatchType    string
}

func ScoreCandidate(input ScoringInput, intent Intent, currentCtx appctx.CurrentContext) ScoredResult {
	textScore := computeTextRelevance(intent, input.Command, input.FTSRank)
	intentScore := computeIntentMatch(intent, input.Command)
	freqScore := computeFrequencyScore(input.Command.Frequency, input.MaxFrequency)
	ctxScore := computeContextScore(input.Contexts, currentCtx)
	recencyScore := computeRecencyScore(input.Command.LastSeen)

	finalScore := textScore*WeightText +
		intentScore*WeightIntent +
		freqScore*WeightFrequency +
		ctxScore*WeightContext +
		recencyScore*WeightRecency

	confidence := finalScore * 100
	if confidence > 99 {
		confidence = 99
	}
	if confidence < 1 {
		confidence = 1
	}

	return ScoredResult{
		Command:        input.Command,
		FinalScore:     finalScore,
		Confidence:     confidence,
		TextScore:      textScore,
		IntentScore:    intentScore,
		FrequencyScore: freqScore,
		ContextScore:   ctxScore,
		RecencyScore:   recencyScore,
		MatchType:      input.MatchType,
	}
}

func computeTextRelevance(intent Intent, cmd vault.Command, ftsRank float64) float64 {
	// FTS5 rank component (normalized, rank is negative — lower is better)
	ftsScore := 0.0
	if ftsRank != 0 {
		absRank := math.Abs(ftsRank)
		ftsScore = absRank / (absRank + 1.0)
	}

	// Token overlap: what fraction of query tokens appear in the command text
	cmdLower := strings.ToLower(cmd.Raw + " " + cmd.Binary + " " + cmd.Subcommand + " " + cmd.Category)
	matchCount := 0
	for _, token := range intent.Tokens {
		if strings.Contains(cmdLower, token) {
			matchCount++
		}
	}
	tokenOverlap := 0.0
	if len(intent.Tokens) > 0 {
		tokenOverlap = float64(matchCount) / float64(len(intent.Tokens))
	}

	// Trigram similarity between query and command
	trigramScore := trigramSimilarity(strings.Join(intent.Tokens, " "), cmd.Raw)

	return clamp(ftsScore*0.4 + tokenOverlap*0.4 + trigramScore*0.2)
}

func computeIntentMatch(intent Intent, cmd vault.Command) float64 {
	if len(intent.ExpandedTokens) == 0 {
		return 0
	}

	cmdLower := strings.ToLower(cmd.Raw + " " + cmd.Binary + " " + cmd.Subcommand + " " + cmd.Category)

	matchCount := 0
	for _, token := range intent.ExpandedTokens {
		if strings.Contains(cmdLower, token) {
			matchCount++
		}
	}

	expandedOverlap := float64(matchCount) / float64(len(intent.ExpandedTokens))

	// Bonus if the command category matches the detected intent category
	categoryBonus := 0.0
	if intent.Category != "" && strings.EqualFold(cmd.Category, intent.Category) {
		categoryBonus = 0.3
	}

	// Bonus if the action matches what the binary does
	actionBonus := 0.0
	if intent.Action != "" {
		actionBonus = actionMatchBonus(intent.Action, cmd.Binary, cmd.Subcommand)
	}

	return clamp(expandedOverlap*0.5 + categoryBonus + actionBonus*0.2)
}

func actionMatchBonus(action, binary, subcommand string) float64 {
	binaryActions := map[string][]string{
		"find":      {"find"},
		"rm":        {"delete"},
		"mv":        {"move"},
		"cp":        {"copy"},
		"kill":      {"stop"},
		"ssh":       {"connect"},
		"tar":       {"archive"},
		"zip":       {"archive"},
		"unzip":     {"archive"},
		"grep":      {"find"},
		"chmod":     {"modify"},
		"chown":     {"modify"},
		"mkdir":     {"create"},
		"touch":     {"create"},
		"curl":      {"connect", "show"},
		"wget":      {"connect", "show"},
		"systemctl": {"run", "stop", "monitor"},
	}

	if actions, ok := binaryActions[binary]; ok {
		for _, a := range actions {
			if a == action {
				return 1.0
			}
		}
	}

	subLower := strings.ToLower(subcommand)
	subActions := map[string]string{
		"delete": "delete", "remove": "delete", "rm": "delete", "prune": "delete",
		"create": "create", "new": "create", "init": "create", "add": "create",
		"list": "show", "ls": "show", "get": "show", "show": "show", "status": "show", "info": "show",
		"start": "run", "run": "run", "up": "run", "exec": "run",
		"stop": "stop", "kill": "stop", "down": "stop",
		"install": "install", "uninstall": "install", "update": "install", "upgrade": "install",
		"push": "copy", "pull": "copy", "clone": "copy", "cp": "copy",
		"log": "monitor", "logs": "monitor", "watch": "monitor",
		"reset": "undo", "revert": "undo", "checkout": "undo", "restore": "undo",
	}
	for keyword, mappedAction := range subActions {
		if strings.Contains(subLower, keyword) && mappedAction == action {
			return 1.0
		}
	}

	return 0.0
}

func computeFrequencyScore(frequency, maxFrequency int) float64 {
	if maxFrequency <= 0 {
		maxFrequency = 1
	}
	// Log-normalized frequency to prevent power-law dominance
	return math.Log1p(float64(frequency)) / math.Log1p(float64(maxFrequency))
}

func computeContextScore(contexts []vault.Context, current appctx.CurrentContext) float64 {
	if len(contexts) == 0 {
		return 0.2 // small base score for knowledge base entries without context
	}

	bestScore := 0.0
	for _, ctx := range contexts {
		score := 0.0

		// Same git repo
		if current.Git.IsRepo && ctx.GitRepo != "" && strings.EqualFold(ctx.GitRepo, current.Git.RepoName) {
			score += 0.35
		}

		// Same project type
		if current.Project.Type != "" && ctx.ProjectType != "" && strings.EqualFold(ctx.ProjectType, current.Project.Type) {
			score += 0.25
		}

		// Same or child directory
		if current.Cwd != "" && ctx.Cwd != "" {
			if ctx.Cwd == current.Cwd {
				score += 0.25
			} else if strings.HasPrefix(current.Cwd, ctx.Cwd) || strings.HasPrefix(ctx.Cwd, current.Cwd) {
				score += 0.15
			}
		}

		if score > bestScore {
			bestScore = score
		}
	}

	return clamp(bestScore)
}

func computeRecencyScore(lastSeen time.Time) float64 {
	if lastSeen.IsZero() {
		return 0.1
	}

	hoursSince := time.Since(lastSeen).Hours()

	switch {
	case hoursSince < 1:
		return 1.0
	case hoursSince < 24:
		return 0.9
	case hoursSince < 24*7:
		return 0.7
	case hoursSince < 24*30:
		return 0.4
	case hoursSince < 24*90:
		return 0.2
	default:
		return 0.1
	}
}

func trigramSimilarity(a, b string) float64 {
	if a == "" || b == "" {
		return 0
	}

	a = strings.ToLower(a)
	b = strings.ToLower(b)

	if len(a) < 3 || len(b) < 3 {
		if strings.Contains(b, a) || strings.Contains(a, b) {
			shorter := len(a)
			if len(b) < shorter {
				shorter = len(b)
			}
			longer := len(a)
			if len(b) > longer {
				longer = len(b)
			}
			return float64(shorter) / float64(longer)
		}
		if a == b {
			return 1.0
		}
		return 0
	}

	triA := trigrams(a)
	triB := trigrams(b)

	intersection := 0
	for tri := range triA {
		if triB[tri] {
			intersection++
		}
	}

	union := len(triA) + len(triB) - intersection
	if union == 0 {
		return 0
	}

	return float64(intersection) / float64(union)
}

func trigrams(s string) map[string]bool {
	tris := make(map[string]bool)
	for i := 0; i <= len(s)-3; i++ {
		tris[s[i:i+3]] = true
	}
	return tris
}

func clamp(v float64) float64 {
	if v < 0 {
		return 0
	}
	if v > 1 {
		return 1
	}
	return v
}

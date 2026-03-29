package intelligence

import (
	"sort"

	appctx "github.com/Om-Rohilla/recall/internal/context"
	"github.com/Om-Rohilla/recall/internal/vault"
)

type SearchOptions struct {
	Limit        int
	VaultOnly    bool
	KBOnly       bool
	Category     string
	KBPath       string
}

// Search runs the full 4-stage intelligence pipeline:
// 1. Intent extraction (tokenize, normalize, expand synonyms)
// 2. Candidate fetch (FTS5 on vault + knowledge base)
// 3. Multi-signal scoring
// 4. Rank and return
func Search(store *vault.Store, query string, currentCtx appctx.CurrentContext, opts SearchOptions) ([]vault.SearchResult, error) {
	if opts.Limit <= 0 {
		opts.Limit = 10
	}

	// Stage 1: Intent extraction
	intent := ExtractIntent(query)
	ftsQuery := BuildFTSQuery(intent)
	if ftsQuery == "" {
		return nil, nil
	}

	maxFreq, _ := store.GetMaxFrequency()
	fetchLimit := opts.Limit * 5 // fetch more candidates for better scoring

	var scored []ScoredResult

	// Stage 2a: Fetch vault candidates
	if !opts.KBOnly {
		vaultResults, err := store.SearchFTS5(ftsQuery, fetchLimit)
		if err == nil {
			for _, r := range vaultResults {
				contexts, _ := store.GetContextsForCommand(r.Command.ID)
				input := ScoringInput{
					Command:      r.Command,
					Contexts:     contexts,
					FTSRank:      r.Score,
					MaxFrequency: maxFreq,
					MatchType:    "vault",
				}
				scored = append(scored, ScoreCandidate(input, intent, currentCtx))
			}
		}
	}

	// Stage 2b: Fetch knowledge base candidates
	if !opts.VaultOnly {
		if opts.KBPath != "" {
			LoadKnowledgeBase(store, opts.KBPath)
		}

		kbResults, err := store.SearchKnowledgeFTS5(ftsQuery, fetchLimit)
		if err == nil {
			for _, k := range kbResults {
				cmd := vault.Command{
					Raw:      k.Command,
					Binary:   extractBinary(k.Command),
					Category: k.Category,
				}
				input := ScoringInput{
					Command:      cmd,
					FTSRank:      1.0, // neutral FTS rank for knowledge
					MaxFrequency: maxFreq,
					MatchType:    "knowledge",
				}
				scored = append(scored, ScoreCandidate(input, intent, currentCtx))
			}
		}
	}

	// Stage 3: Sort by final score (descending)
	sort.Slice(scored, func(i, j int) bool {
		return scored[i].FinalScore > scored[j].FinalScore
	})

	// Deduplicate by raw command
	scored = dedup(scored)

	// Filter by category if specified
	if opts.Category != "" {
		var filtered []ScoredResult
		for _, s := range scored {
			if s.Command.Category == opts.Category {
				filtered = append(filtered, s)
			}
		}
		scored = filtered
	}

	// Stage 4: Convert to SearchResult and limit
	if len(scored) > opts.Limit {
		scored = scored[:opts.Limit]
	}

	var results []vault.SearchResult
	for _, s := range scored {
		results = append(results, vault.SearchResult{
			Command:    s.Command,
			Score:      s.FinalScore,
			Confidence: s.Confidence,
			MatchType:  s.MatchType,
		})
	}

	return results, nil
}

func dedup(scored []ScoredResult) []ScoredResult {
	seen := make(map[string]bool)
	var unique []ScoredResult
	for _, s := range scored {
		if !seen[s.Command.Raw] {
			seen[s.Command.Raw] = true
			unique = append(unique, s)
		}
	}
	return unique
}

func extractBinary(raw string) string {
	for i, r := range raw {
		if r == ' ' || r == '\t' {
			return raw[:i]
		}
	}
	return raw
}

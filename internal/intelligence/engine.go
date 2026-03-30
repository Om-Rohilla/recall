package intelligence

import (
	"fmt"
	"sort"
	"strings"
	"sync"

	appctx "github.com/Om-Rohilla/recall/internal/context"
	"github.com/Om-Rohilla/recall/internal/vault"
	"github.com/Om-Rohilla/recall/pkg/logging"
)

var (
	kbMu     sync.Mutex
	kbLoaded bool
)

// ResetKnowledgeBase allows the knowledge base to be reloaded on next search.
// Call this after updating the knowledge base file on disk.
func ResetKnowledgeBase() {
	kbMu.Lock()
	kbLoaded = false
	kbMu.Unlock()
}

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
	fetchLimit := opts.Limit * 5

	var scored []ScoredResult
	var vaultErr, kbErr error

	// Stage 2a: Fetch vault candidates
	if !opts.KBOnly {
		vaultResults, err := store.SearchFTS5(ftsQuery, fetchLimit)
		if err != nil {
			vaultErr = err
		} else {
			cmdIDs := make([]int64, 0, len(vaultResults))
			for _, r := range vaultResults {
				cmdIDs = append(cmdIDs, r.Command.ID)
			}
			ctxMap := batchGetContexts(store, cmdIDs)

			for _, r := range vaultResults {
				input := ScoringInput{
					Command:      r.Command,
					Contexts:     ctxMap[r.Command.ID],
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
			kbMu.Lock()
			if !kbLoaded {
				LoadKnowledgeBase(store, opts.KBPath)
				kbLoaded = true
			}
			kbMu.Unlock()
		}

		kbResults, err := store.SearchKnowledgeFTS5(ftsQuery, fetchLimit)
		if err != nil {
			kbErr = err
		} else {
			for _, k := range kbResults {
				cmd := vault.Command{
					Raw:      k.Command,
					Binary:   extractBinary(k.Command),
					Category: k.Category,
				}
				input := ScoringInput{
					Command:      cmd,
					FTSRank:      1.0,
					MaxFrequency: maxFreq,
					MatchType:    "knowledge",
				}
				scored = append(scored, ScoreCandidate(input, intent, currentCtx))
			}
		}
	}

	log := logging.Get()
	if vaultErr != nil {
		log.Warn("vault FTS5 search failed", "error", vaultErr, "query", ftsQuery)
	}
	if kbErr != nil {
		log.Warn("knowledge base FTS5 search failed", "error", kbErr, "query", ftsQuery)
	}
	if len(scored) == 0 && (vaultErr != nil || kbErr != nil) {
		if vaultErr != nil {
			return nil, fmt.Errorf("vault search failed: %w", vaultErr)
		}
		return nil, fmt.Errorf("knowledge base search failed: %w", kbErr)
	}
	log.Debug("search completed", "query", query, "results", len(scored))

	// Stage 3: Sort by final score (descending)
	sort.Slice(scored, func(i, j int) bool {
		return scored[i].FinalScore > scored[j].FinalScore
	})

	// Deduplicate by raw command
	scored = dedup(scored)

	if opts.Category != "" {
		catLower := strings.ToLower(opts.Category)
		var filtered []ScoredResult
		for _, s := range scored {
			if strings.ToLower(s.Command.Category) == catLower {
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
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	for i, r := range raw {
		if r == ' ' || r == '\t' {
			return raw[:i]
		}
	}
	return raw
}

// batchGetContexts fetches contexts for multiple command IDs in a single query,
// eliminating the N+1 query pattern that previously ran one query per candidate.
func batchGetContexts(store *vault.Store, ids []int64) map[int64][]vault.Context {
	result := make(map[int64][]vault.Context, len(ids))
	if len(ids) == 0 {
		return result
	}

	const batchSize = 50
	for start := 0; start < len(ids); start += batchSize {
		end := start + batchSize
		if end > len(ids) {
			end = len(ids)
		}
		batch := ids[start:end]

		ctxs, err := store.GetContextsForCommandBatch(batch)
		if err != nil {
			for _, id := range batch {
				c, _ := store.GetContextsForCommand(id)
				result[id] = c
			}
			continue
		}
		for cmdID, c := range ctxs {
			result[cmdID] = c
		}
	}
	return result
}


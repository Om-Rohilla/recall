package intelligence

import (
	"testing"
	"time"

	appctx "github.com/Om-Rohilla/recall/internal/context"
	"github.com/Om-Rohilla/recall/internal/vault"
)

func TestScoreCandidate_TextRelevance(t *testing.T) {
	intent := Intent{
		RawQuery: "git commit",
		Tokens:   []string{"git", "commit"},
	}
	input := ScoringInput{
		Command: vault.Command{
			Raw:    "git commit -m 'fix bug'",
			Binary: "git",
		},
		MatchType: "vault",
	}

	result := ScoreCandidate(input, intent, appctx.CurrentContext{})
	if result.FinalScore <= 0 {
		t.Errorf("expected positive score for matching command, got %f", result.FinalScore)
	}
	if result.Confidence <= 0 {
		t.Errorf("expected positive confidence, got %f", result.Confidence)
	}
}

func TestScoreCandidate_FrequencyBoost(t *testing.T) {
	intent := Intent{
		RawQuery: "git",
		Tokens:   []string{"git"},
	}

	inputLow := ScoringInput{
		Command: vault.Command{
			Raw:       "git status",
			Binary:    "git",
			Frequency: 1,
		},
		MaxFrequency: 100,
	}
	inputHigh := ScoringInput{
		Command: vault.Command{
			Raw:       "git status",
			Binary:    "git",
			Frequency: 100,
		},
		MaxFrequency: 100,
	}

	scoreLow := ScoreCandidate(inputLow, intent, appctx.CurrentContext{})
	scoreHigh := ScoreCandidate(inputHigh, intent, appctx.CurrentContext{})

	if scoreHigh.FrequencyScore <= scoreLow.FrequencyScore {
		t.Errorf("high frequency (%f) should score higher than low frequency (%f)",
			scoreHigh.FrequencyScore, scoreLow.FrequencyScore)
	}
}

func TestScoreCandidate_RecencyBoost(t *testing.T) {
	intent := Intent{
		RawQuery: "git",
		Tokens:   []string{"git"},
	}

	inputRecent := ScoringInput{
		Command: vault.Command{
			Raw:      "git status",
			Binary:   "git",
			LastSeen: time.Now().Add(-1 * time.Hour),
		},
	}
	inputOld := ScoringInput{
		Command: vault.Command{
			Raw:      "git status",
			Binary:   "git",
			LastSeen: time.Now().Add(-180 * 24 * time.Hour),
		},
	}

	scoreRecent := ScoreCandidate(inputRecent, intent, appctx.CurrentContext{})
	scoreOld := ScoreCandidate(inputOld, intent, appctx.CurrentContext{})

	if scoreRecent.RecencyScore <= scoreOld.RecencyScore {
		t.Errorf("recent command (%f) should have higher recency score than old (%f)",
			scoreRecent.RecencyScore, scoreOld.RecencyScore)
	}
}

func TestScoreCandidate_CategoryMatch(t *testing.T) {
	intent := Intent{
		RawQuery: "docker containers",
		Tokens:   []string{"docker", "containers"},
		Category: "docker",
	}

	inputMatch := ScoringInput{
		Command: vault.Command{
			Raw:      "docker ps -a",
			Binary:   "docker",
			Category: "docker",
		},
	}
	inputNoMatch := ScoringInput{
		Command: vault.Command{
			Raw:      "ls -la",
			Binary:   "ls",
			Category: "filesystem",
		},
	}

	scoreMatch := ScoreCandidate(inputMatch, intent, appctx.CurrentContext{})
	scoreNoMatch := ScoreCandidate(inputNoMatch, intent, appctx.CurrentContext{})

	// The docker command should score higher overall because it matches the query text
	if scoreMatch.FinalScore < scoreNoMatch.FinalScore {
		t.Errorf("category-matching final score (%f) should be >= non-matching (%f)",
			scoreMatch.FinalScore, scoreNoMatch.FinalScore)
	}
}

func TestScoreCandidate_ZeroTokens(t *testing.T) {
	intent := Intent{
		RawQuery: "",
		Tokens:   []string{},
	}
	input := ScoringInput{
		Command: vault.Command{
			Raw:    "git status",
			Binary: "git",
		},
	}

	result := ScoreCandidate(input, intent, appctx.CurrentContext{})
	if result.FinalScore < 0 {
		t.Errorf("should not produce negative score, got %f", result.FinalScore)
	}
}

func TestComputeRecencyScore(t *testing.T) {
	now := time.Now()

	recentScore := computeRecencyScore(now.Add(-1 * time.Hour))
	oldScore := computeRecencyScore(now.Add(-365 * 24 * time.Hour))

	if recentScore <= oldScore {
		t.Errorf("recent (%f) should score higher than old (%f)", recentScore, oldScore)
	}

	// Very recent should be high
	if recentScore < 0.5 {
		t.Errorf("1-hour-old command should have recency >= 0.5, got %f", recentScore)
	}
}

func TestComputeFrequencyScore(t *testing.T) {
	low := computeFrequencyScore(1, 100)
	high := computeFrequencyScore(100, 100)

	if high <= low {
		t.Errorf("high frequency (%f) should score higher than low (%f)", high, low)
	}
}

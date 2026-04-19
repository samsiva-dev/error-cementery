package match

import (
	"context"
	"fmt"

	"github.com/samsiva-dev/error-cemetery/internal/ai"
	"github.com/samsiva-dev/error-cemetery/internal/db"
)

const ftsThreshold = 0.6

type MatchResult struct {
	Burial    db.Burial
	Score     float64
	MatchType string // "exact" | "fts" | "semantic"
}

func Rank(query string, store *db.Store, aiClient *ai.Client, smart bool) ([]MatchResult, error) {
	// Pass 1: exact hash match
	hash := db.HashError(query)
	if burial, err := store.GetByHash(hash); err == nil {
		_ = store.UpdateDigCount(burial.ID)
		return []MatchResult{{Burial: *burial, Score: 1.0, MatchType: "exact"}}, nil
	}

	// Pass 2: FTS BM25 full-text search
	ftsResults, err := ftsSearch(store, query)
	if err != nil {
		return nil, fmt.Errorf("fts search: %w", err)
	}

	if !smart && len(ftsResults) > 0 && ftsResults[0].Score >= ftsThreshold {
		for _, r := range ftsResults {
			_ = store.UpdateDigCount(r.Burial.ID)
		}
		return ftsResults, nil
	}

	// Pass 3: Claude semantic re-ranking (only with --smart or low FTS confidence)
	if smart && aiClient != nil && len(ftsResults) > 0 {
		reranked, err := semanticRerank(query, ftsResults, aiClient)
		if err == nil && len(reranked) > 0 {
			for _, r := range reranked {
				_ = store.UpdateDigCount(r.Burial.ID)
			}
			return reranked, nil
		}
	}

	// Return FTS results even if below threshold
	for _, r := range ftsResults {
		_ = store.UpdateDigCount(r.Burial.ID)
	}
	return ftsResults, nil
}

func ftsSearch(store *db.Store, query string) ([]MatchResult, error) {
	burials, err := store.FTSSearch(escapeFTS(query), 10)
	if err != nil {
		// FTS failed (e.g. special chars) — return empty, not error
		return nil, nil
	}

	results := make([]MatchResult, len(burials))
	for i, b := range burials {
		results[i] = MatchResult{
			Burial:    b,
			Score:     scoreByRank(i, len(burials)),
			MatchType: "fts",
		}
	}
	return results, nil
}

func semanticRerank(query string, candidates []MatchResult, aiClient *ai.Client) ([]MatchResult, error) {
	cands := make([]ai.RankCandidate, len(candidates))
	for i, c := range candidates {
		cands[i] = ai.RankCandidate{ID: c.Burial.ID, ErrorText: c.Burial.ErrorText}
	}

	ranked, err := aiClient.RankCandidates(context.Background(), query, cands)
	if err != nil {
		return nil, err
	}

	scoreMap := map[int64]float64{}
	for _, r := range ranked {
		scoreMap[r.ID] = r.Score
	}

	results := make([]MatchResult, 0, len(candidates))
	for _, c := range candidates {
		score, ok := scoreMap[c.Burial.ID]
		if !ok {
			score = c.Score
		}
		results = append(results, MatchResult{
			Burial:    c.Burial,
			Score:     score,
			MatchType: "semantic",
		})
	}

	sortByScore(results)
	if len(results) > 5 {
		results = results[:5]
	}
	return results, nil
}

func sortByScore(results []MatchResult) {
	for i := 1; i < len(results); i++ {
		for j := i; j > 0 && results[j].Score > results[j-1].Score; j-- {
			results[j], results[j-1] = results[j-1], results[j]
		}
	}
}

func scoreByRank(rank, total int) float64 {
	if total == 0 {
		return 0
	}
	return 1.0 - (float64(rank) / float64(total))
}

// escapeFTS makes a user query safe for FTS5 MATCH.
// We wrap the whole query in double quotes to treat it as a phrase, falling
// back to individual terms if that fails.
func escapeFTS(q string) string {
	return fmt.Sprintf(`"%s"`, q)
}

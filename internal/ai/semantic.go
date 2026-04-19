package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"strings"

	anthropic "github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
)

type Client struct {
	client anthropic.Client
	model  string
}

func NewClient(apiKey, model string) *Client {
	return &Client{
		client: anthropic.NewClient(option.WithAPIKey(apiKey)),
		model:  model,
	}
}

type RankCandidate struct {
	ID        int64
	ErrorText string
}

type RankResult struct {
	ID    int64   `json:"id"`
	Score float64 `json:"score"`
}

const rankPrompt = `You are a search engine for a developer's personal error database.

Query error:
<query>%s</query>

Candidate errors (JSON array, each with id and error_text):
<candidates>%s</candidates>

Return a JSON array of objects {"id": <number>, "score": <0.0-1.0>} where score is semantic similarity to the query.
Only return the JSON array, no other text.`

func (c *Client) RankCandidates(ctx context.Context, query string, candidates []RankCandidate) ([]RankResult, error) {
	if len(candidates) == 0 {
		return nil, nil
	}

	type candJSON struct {
		ID        int64  `json:"id"`
		ErrorText string `json:"error_text"`
	}
	cands := make([]candJSON, len(candidates))
	for i, ca := range candidates {
		cands[i] = candJSON{ID: ca.ID, ErrorText: ca.ErrorText}
	}
	candsJSON, _ := json.Marshal(cands)

	prompt := fmt.Sprintf(rankPrompt, query, string(candsJSON))

	msg, err := c.client.Messages.New(ctx, anthropic.MessageNewParams{
		Model:     anthropic.Model(c.model),
		MaxTokens: 1024,
		Messages: []anthropic.MessageParam{
			anthropic.NewUserMessage(anthropic.NewTextBlock(prompt)),
		},
	})
	if err != nil {
		return nil, fmt.Errorf("claude rank: %w", err)
	}

	var raw string
	for _, block := range msg.Content {
		if block.Type == "text" {
			raw = block.AsText().Text
			break
		}
	}

	raw = strings.TrimSpace(raw)
	var results []RankResult
	if err := json.Unmarshal([]byte(raw), &results); err != nil {
		return nil, fmt.Errorf("parse rank response: %w", err)
	}
	return results, nil
}

func CosineSimilarity(a, b []float32) float64 {
	if len(a) != len(b) {
		return 0
	}
	var dot, normA, normB float64
	for i := range a {
		dot += float64(a[i]) * float64(b[i])
		normA += float64(a[i]) * float64(a[i])
		normB += float64(b[i]) * float64(b[i])
	}
	if normA == 0 || normB == 0 {
		return 0
	}
	return dot / (math.Sqrt(normA) * math.Sqrt(normB))
}

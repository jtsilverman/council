package strategy

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/jtsilverman/council/internal/council"
	"github.com/jtsilverman/council/internal/provider"
)

// VoteStrategy implements: parallel review -> chair aggregation (no debate round).
// Cheaper and faster than debate. Good for factual questions.
type VoteStrategy struct{}

func (s *VoteStrategy) Run(ctx context.Context, c *council.Council, query string, p *council.Providers) (*council.Deliberation, error) {
	delib := &council.Deliberation{}

	// Phase 1: Independent review (parallel)
	fmt.Fprintf(os.Stderr, "Phase 1: Independent review (%d members)...\n", len(c.Members))
	ds := &DebateStrategy{}
	reviewResponses, err := ds.parallelReview(ctx, c.Members, query, p)
	if err != nil {
		return nil, fmt.Errorf("review phase: %w", err)
	}
	delib.Rounds = append(delib.Rounds, council.Round{
		Phase:     "review",
		Responses: reviewResponses,
	})

	// Phase 2: Chair synthesis (no debate)
	fmt.Fprintf(os.Stderr, "Phase 2: Chair aggregation...\n")
	var b strings.Builder
	b.WriteString(fmt.Sprintf("Original query:\n%s\n\n", query))
	b.WriteString("=== MEMBER RESPONSES ===\n\n")
	for _, r := range reviewResponses {
		b.WriteString(fmt.Sprintf("--- %s ---\n%s\n\n", r.Member, r.Content))
	}
	b.WriteString(`=== YOUR TASK ===

Synthesize a final response from the member responses above.
Identify consensus points (majority agreement) and note any disagreements.
Produce a clear, decisive answer. Prioritize by impact.`)

	chairProvider := p.Default
	resp, err := chairProvider.Complete(ctx, provider.CompletionRequest{
		SystemPrompt: c.Chair.Persona,
		UserPrompt:   b.String(),
		Model:        c.Chair.Model,
		MaxTokens:    4096,
	})
	if err != nil {
		return nil, fmt.Errorf("synthesis: %w", err)
	}

	delib.Synthesis = council.Response{
		Member:  c.Chair.Name,
		Content: resp.Content,
		Tokens:  council.TokenUsage{Input: resp.Tokens.Input, Output: resp.Tokens.Output, Cost: resp.Tokens.Cost},
		Latency: resp.Latency,
	}

	return delib, nil
}

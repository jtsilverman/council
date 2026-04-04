package strategy

import (
	"context"
	"fmt"
	"os"

	"github.com/jtsilverman/council/internal/council"
	"github.com/jtsilverman/council/internal/provider"
	"github.com/jtsilverman/council/internal/review"
)

// VoteStrategy implements: parallel review -> chair aggregation (no debate round).
// Cheaper and faster than debate. Good for factual questions.
type VoteStrategy struct{}

func (s *VoteStrategy) Run(ctx context.Context, c *council.Council, query string, p *council.Providers) (*council.Deliberation, error) {
	delib := &council.Deliberation{}

	// Phase 1: Independent review (parallel)
	fmt.Fprintf(os.Stderr, "Phase 1: Independent review (%d members)...\n", len(c.Members))
	ds := &DebateStrategy{}
	profile := provider.DefaultProfile()
	reviewResponses, err := ds.parallelReview(ctx, c.Members, query, profile, p)
	if err != nil {
		return nil, fmt.Errorf("review phase: %w", err)
	}
	delib.Rounds = append(delib.Rounds, council.Round{
		Phase:     "review",
		Responses: reviewResponses,
	})

	// Parse review responses into digests
	reviewDigests := make([]review.ReviewDigest, len(reviewResponses))
	for i, r := range reviewResponses {
		reviewDigests[i] = review.ParseDigest(r.Content)
	}

	// Phase 2: Chair synthesis (no debate)
	fmt.Fprintf(os.Stderr, "Phase 2: Chair aggregation...\n")
	synthesisPrompt := BuildSynthesisPrompt(profile, query, reviewDigests)

	chairProvider := p.Default
	resp, err := chairProvider.Complete(ctx, provider.CompletionRequest{
		SystemPrompt: c.Chair.Persona,
		UserPrompt:   synthesisPrompt,
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

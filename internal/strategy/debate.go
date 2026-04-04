package strategy

import (
	"context"
	"fmt"
	"os"
	"sync"

	"github.com/jtsilverman/council/internal/council"
	"github.com/jtsilverman/council/internal/provider"
	"github.com/jtsilverman/council/internal/review"
)

// DebateStrategy implements: parallel review -> debate -> chair synthesis.
type DebateStrategy struct{}

func (s *DebateStrategy) Run(ctx context.Context, c *council.Council, query string, p *council.Providers) (*council.Deliberation, error) {
	delib := &council.Deliberation{}

	// Phase 1: Independent review (parallel)
	fmt.Fprintf(os.Stderr, "Phase 1: Independent review (%d members)...\n", len(c.Members))
	reviewResponses, err := s.parallelReview(ctx, c.Members, query, p)
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

	// Phase 2: Debate (parallel)
	fmt.Fprintf(os.Stderr, "Phase 2: Debate...\n")
	debateResponses, err := s.parallelDebateWithDigests(ctx, c.Members, query, reviewDigests, p)
	if err != nil {
		return nil, fmt.Errorf("debate phase: %w", err)
	}
	delib.Rounds = append(delib.Rounds, council.Round{
		Phase:     "debate",
		Responses: debateResponses,
	})

	// Parse debate responses into digests
	debateDigests := make([]review.ReviewDigest, len(debateResponses))
	for i, r := range debateResponses {
		debateDigests[i] = review.ParseDigest(r.Content)
	}

	// Phase 3: Chair synthesis
	fmt.Fprintf(os.Stderr, "Phase 3: Chair synthesis...\n")
	allDigests := make([]review.ReviewDigest, 0, len(reviewDigests)+len(debateDigests))
	allDigests = append(allDigests, reviewDigests...)
	allDigests = append(allDigests, debateDigests...)
	chairProfile := provider.ProfileFor(p.Default)
	synthesisPrompt := BuildSynthesisPrompt(chairProfile, query, allDigests)
	chairProvider := p.Default // Chair always uses default provider
	resp, err := chairProvider.Complete(ctx, provider.CompletionRequest{
		SystemPrompt: c.Chair.Persona,
		UserPrompt:   synthesisPrompt,
		Model:        c.Chair.Model,
		MaxTokens:    4096,
	})
	if err != nil {
		return nil, fmt.Errorf("synthesis phase: %w", err)
	}

	delib.Synthesis = council.Response{
		Member:  c.Chair.Name,
		Content: resp.Content,
		Tokens:  council.TokenUsage{Input: resp.Tokens.Input, Output: resp.Tokens.Output, Cost: resp.Tokens.Cost},
		Latency: resp.Latency,
	}

	return delib, nil
}

func (s *DebateStrategy) parallelReview(ctx context.Context, members []council.Member, query string, p *council.Providers) ([]council.Response, error) {
	responses := make([]council.Response, len(members))
	var wg sync.WaitGroup
	var mu sync.Mutex
	var firstErr error

	for i, m := range members {
		wg.Add(1)
		go func(idx int, member council.Member) {
			defer wg.Done()
			prov := p.For(idx)
			profile := provider.ProfileFor(prov)
			prompt := BuildReviewPrompt(profile, query)
			resp, err := prov.Complete(ctx, provider.CompletionRequest{
				SystemPrompt: member.Persona,
				UserPrompt:   prompt,
				Model:        member.Model,
				MaxTokens:    4096,
			})
			if err != nil {
				mu.Lock()
				if firstErr == nil {
					firstErr = fmt.Errorf("%s: %w", member.Name, err)
				}
				mu.Unlock()
				return
			}
			responses[idx] = council.Response{
				Member:  member.Name,
				Content: resp.Content,
				Tokens:  council.TokenUsage{Input: resp.Tokens.Input, Output: resp.Tokens.Output, Cost: resp.Tokens.Cost},
				Latency: resp.Latency,
			}
			fmt.Fprintf(os.Stderr, "  ✓ %s\n", member.Name)
		}(i, m)
	}

	wg.Wait()
	if firstErr != nil {
		return nil, firstErr
	}
	return responses, nil
}

// parallelDebateWithDigests runs the debate phase using structured digests from phase 1.
func (s *DebateStrategy) parallelDebateWithDigests(ctx context.Context, members []council.Member, query string, digests []review.ReviewDigest, p *council.Providers) ([]council.Response, error) {
	responses := make([]council.Response, len(members))
	var wg sync.WaitGroup
	var mu sync.Mutex
	var firstErr error

	for i, m := range members {
		wg.Add(1)
		go func(idx int, member council.Member) {
			defer wg.Done()

			prov := p.For(idx)
			profile := provider.ProfileFor(prov)
			prompt := BuildDebatePrompt(profile, query, digests)
			resp, err := prov.Complete(ctx, provider.CompletionRequest{
				SystemPrompt: member.Persona,
				UserPrompt:   prompt,
				Model:        member.Model,
				MaxTokens:    4096,
			})
			if err != nil {
				mu.Lock()
				if firstErr == nil {
					firstErr = fmt.Errorf("%s debate: %w", member.Name, err)
				}
				mu.Unlock()
				return
			}
			responses[idx] = council.Response{
				Member:  member.Name,
				Content: resp.Content,
				Tokens:  council.TokenUsage{Input: resp.Tokens.Input, Output: resp.Tokens.Output, Cost: resp.Tokens.Cost},
				Latency: resp.Latency,
			}
			fmt.Fprintf(os.Stderr, "  ✓ %s (debate)\n", member.Name)
		}(i, m)
	}

	wg.Wait()
	if firstErr != nil {
		return nil, firstErr
	}
	return responses, nil
}


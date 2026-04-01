package strategy

import (
	"context"
	"fmt"
	"os"
	"strings"
	"sync"

	"github.com/jtsilverman/council/internal/council"
	"github.com/jtsilverman/council/internal/provider"
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

	// Phase 2: Debate (parallel)
	fmt.Fprintf(os.Stderr, "Phase 2: Debate...\n")
	debatePrompt := s.buildDebateContext(reviewResponses)
	debateResponses, err := s.parallelDebate(ctx, c.Members, query, debatePrompt, p)
	if err != nil {
		return nil, fmt.Errorf("debate phase: %w", err)
	}
	delib.Rounds = append(delib.Rounds, council.Round{
		Phase:     "debate",
		Responses: debateResponses,
	})

	// Phase 3: Chair synthesis
	fmt.Fprintf(os.Stderr, "Phase 3: Chair synthesis...\n")
	synthesisPrompt := s.buildSynthesisPrompt(query, reviewResponses, debateResponses)
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
			resp, err := prov.Complete(ctx, provider.CompletionRequest{
				SystemPrompt: member.Persona,
				UserPrompt:   query,
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

func (s *DebateStrategy) buildDebateContext(reviews []council.Response) string {
	var b strings.Builder
	b.WriteString("Here are the independent findings from each council member:\n\n")
	for _, r := range reviews {
		b.WriteString(fmt.Sprintf("=== %s ===\n%s\n\n", r.Member, r.Content))
	}
	return b.String()
}

func (s *DebateStrategy) parallelDebate(ctx context.Context, members []council.Member, query string, debateContext string, p *council.Providers) ([]council.Response, error) {
	responses := make([]council.Response, len(members))
	var wg sync.WaitGroup
	var mu sync.Mutex
	var firstErr error

	debateInstructions := `Now review the other council members' findings above.

Your task:
1. CHALLENGE any finding you disagree with. State your specific technical reason.
2. SUPPORT any finding you think is especially important. Explain why.
3. ADD anything the other members missed that falls within your expertise.

Stay in character. Only challenge findings where you have specific technical grounds.
Do not repeat or summarize. Focus on disagreements and additions.`

	for i, m := range members {
		wg.Add(1)
		go func(idx int, member council.Member) {
			defer wg.Done()

			prompt := fmt.Sprintf("Original query:\n%s\n\n%s\n%s", query, debateContext, debateInstructions)

			prov := p.For(idx)
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

func (s *DebateStrategy) buildSynthesisPrompt(query string, reviews, debates []council.Response) string {
	var b strings.Builder
	b.WriteString(fmt.Sprintf("Original query:\n%s\n\n", query))

	b.WriteString("=== PHASE 1: INDEPENDENT REVIEWS ===\n\n")
	for _, r := range reviews {
		b.WriteString(fmt.Sprintf("--- %s ---\n%s\n\n", r.Member, r.Content))
	}

	b.WriteString("=== PHASE 2: DEBATE ===\n\n")
	for _, r := range debates {
		b.WriteString(fmt.Sprintf("--- %s ---\n%s\n\n", r.Member, r.Content))
	}

	b.WriteString(`=== YOUR TASK ===

Synthesize a final response based on the reviews and debate above.

Rules:
- Points with consensus support (2+ members agree) are high confidence. Include them.
- Points that were challenged with valid reasoning should be downgraded or dropped.
- Points that were supported during debate should be highlighted.
- Do not introduce new points that no member raised.
- Be decisive. Prioritize by impact.
- Produce a clear, well-organized final answer.`)

	return b.String()
}

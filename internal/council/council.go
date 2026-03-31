package council

import (
	"context"
	"time"

	"github.com/jtsilverman/council/internal/provider"
)

// Strategy defines how a council deliberates.
type Strategy interface {
	Run(ctx context.Context, c *Council, query string, p provider.Provider) (*Deliberation, error)
}

// Run executes a council deliberation with the given strategy.
func Run(ctx context.Context, c *Council, query string, p provider.Provider, strat Strategy) (*Deliberation, error) {
	start := time.Now()

	delib, err := strat.Run(ctx, c, query, p)
	if err != nil {
		return nil, err
	}

	delib.Query = query
	delib.Council = c.Name
	delib.Strategy = c.Strategy
	delib.Duration = time.Since(start)

	// Sum up tokens and cost
	for _, round := range delib.Rounds {
		for _, resp := range round.Responses {
			delib.TotalTokens.Input += resp.Tokens.Input
			delib.TotalTokens.Output += resp.Tokens.Output
			delib.TotalCost += resp.Tokens.Cost
		}
	}
	delib.TotalTokens.Input += delib.Synthesis.Tokens.Input
	delib.TotalTokens.Output += delib.Synthesis.Tokens.Output
	delib.TotalCost += delib.Synthesis.Tokens.Cost

	return delib, nil
}

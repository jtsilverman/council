package strategy

import "github.com/jtsilverman/council/internal/council"

// Get returns a strategy by name. Defaults to debate.
func Get(name string) council.Strategy {
	switch name {
	case "vote":
		return &VoteStrategy{}
	case "debate":
		return &DebateStrategy{}
	default:
		return &DebateStrategy{}
	}
}

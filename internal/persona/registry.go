package persona

import (
	"fmt"

	"github.com/jtsilverman/council/internal/council"
)

var registry = map[string]*council.Council{}

func register(c *council.Council) {
	registry[c.Name] = c
}

// GetCouncil returns a council by name. Returns a copy so callers can modify it.
func GetCouncil(name string) (*council.Council, error) {
	c, ok := registry[name]
	if !ok {
		return nil, fmt.Errorf("unknown council %q. Run 'council list' to see available councils", name)
	}
	// Return a copy
	copy := *c
	members := make([]council.Member, len(c.Members))
	for i, m := range c.Members {
		members[i] = m
	}
	copy.Members = members
	return &copy, nil
}

// ListCouncils returns all registered councils.
func ListCouncils() []*council.Council {
	var out []*council.Council
	for _, c := range registry {
		out = append(out, c)
	}
	return out
}

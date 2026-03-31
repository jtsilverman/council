package council

import "time"

// Deliberation is the full trace of a council run.
type Deliberation struct {
	Query       string        `json:"query"`
	Council     string        `json:"council"`
	Strategy    string        `json:"strategy"`
	Rounds      []Round       `json:"rounds"`
	Synthesis   Response      `json:"synthesis"`
	TotalTokens TokenUsage    `json:"total_tokens"`
	TotalCost   float64       `json:"total_cost"`
	Duration    time.Duration `json:"duration_ms"`
}

// Round represents one phase of deliberation.
type Round struct {
	Phase     string     `json:"phase"` // "review", "debate", "synthesis"
	Responses []Response `json:"responses"`
}

// Response is a single member's response.
type Response struct {
	Member  string        `json:"member"`
	Content string        `json:"content"`
	Tokens  TokenUsage    `json:"tokens"`
	Latency time.Duration `json:"latency_ms"`
}

// TokenUsage tracks API token usage.
type TokenUsage struct {
	Input  int     `json:"input"`
	Output int     `json:"output"`
	Cost   float64 `json:"cost_usd"`
}

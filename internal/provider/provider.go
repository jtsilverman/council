package provider

import (
	"context"
	"time"
)

// Provider is the interface for LLM API providers.
type Provider interface {
	Complete(ctx context.Context, req CompletionRequest) (*CompletionResponse, error)
	Name() string
}

// CompletionRequest is the input to a provider.
type CompletionRequest struct {
	SystemPrompt string
	UserPrompt   string
	Model        string
	MaxTokens    int
}

// CompletionResponse is the output from a provider.
type CompletionResponse struct {
	Content string
	Tokens  TokenUsage
	Latency time.Duration
}

// TokenUsage tracks API token usage and cost.
type TokenUsage struct {
	Input  int
	Output int
	Cost   float64
}

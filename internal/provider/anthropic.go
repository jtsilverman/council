package provider

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
)

// AnthropicProvider uses the Anthropic API for completions.
type AnthropicProvider struct {
	client *anthropic.Client
}

// NewAnthropicProvider creates an Anthropic provider using ANTHROPIC_API_KEY.
func NewAnthropicProvider() (*AnthropicProvider, error) {
	key := os.Getenv("ANTHROPIC_API_KEY")
	if key == "" {
		return nil, fmt.Errorf("ANTHROPIC_API_KEY not set. Use --api with this env var, or omit --api to use CLI mode ($0)")
	}
	client := anthropic.NewClient(option.WithAPIKey(key))
	return &AnthropicProvider{client: &client}, nil
}

func (p *AnthropicProvider) Name() string { return "anthropic" }

func (p *AnthropicProvider) Complete(ctx context.Context, req CompletionRequest) (*CompletionResponse, error) {
	start := time.Now()

	model := req.Model
	if model == "" {
		model = "claude-sonnet-4-20250514"
	}

	maxTokens := int64(req.MaxTokens)
	if maxTokens == 0 {
		maxTokens = 4096
	}

	params := anthropic.MessageNewParams{
		Model:     model,
		MaxTokens: maxTokens,
		Messages: []anthropic.MessageParam{
			anthropic.NewUserMessage(anthropic.NewTextBlock(req.UserPrompt)),
		},
	}

	if req.SystemPrompt != "" {
		params.System = []anthropic.TextBlockParam{
			{Text: req.SystemPrompt},
		}
	}

	resp, err := p.client.Messages.New(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("anthropic API: %w", err)
	}

	content := ""
	for _, block := range resp.Content {
		if block.Type == "text" {
			content += block.Text
		}
	}

	inputTokens := int(resp.Usage.InputTokens)
	outputTokens := int(resp.Usage.OutputTokens)
	cost := float64(inputTokens)*3.0/1_000_000 + float64(outputTokens)*15.0/1_000_000

	return &CompletionResponse{
		Content: content,
		Latency: time.Since(start),
		Tokens: TokenUsage{
			Input:  inputTokens,
			Output: outputTokens,
			Cost:   cost,
		},
	}, nil
}

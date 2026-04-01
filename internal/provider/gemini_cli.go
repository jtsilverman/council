package provider

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"time"
)

// GeminiCLIProvider uses the `gemini` CLI for completions via Google subscription ($0).
type GeminiCLIProvider struct {
	model string
}

func NewGeminiCLIProvider(model string) *GeminiCLIProvider {
	return &GeminiCLIProvider{model: model}
}

func (p *GeminiCLIProvider) Name() string { return "gemini-cli" }

func (p *GeminiCLIProvider) Complete(ctx context.Context, req CompletionRequest) (*CompletionResponse, error) {
	start := time.Now()

	prompt := req.UserPrompt
	if req.SystemPrompt != "" {
		prompt = fmt.Sprintf("[System: %s]\n\n%s", req.SystemPrompt, req.UserPrompt)
	}

	cmd := exec.CommandContext(ctx, "gemini", prompt)
	cmd.Stdin = bytes.NewBufferString(prompt)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("gemini CLI failed: %w (stderr: %s)", err, stderr.String())
	}

	return &CompletionResponse{
		Content: stdout.String(),
		Latency: time.Since(start),
		Tokens:  TokenUsage{},
	}, nil
}

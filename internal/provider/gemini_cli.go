package provider

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"time"
)

// GeminiCLIProvider uses the `gemini` CLI to run completions.
type GeminiCLIProvider struct{}

// NewGeminiCLIProvider creates a Gemini CLI provider.
func NewGeminiCLIProvider() *GeminiCLIProvider {
	return &GeminiCLIProvider{}
}

func (p *GeminiCLIProvider) Name() string { return "gemini-cli" }

func (p *GeminiCLIProvider) Complete(ctx context.Context, req CompletionRequest) (*CompletionResponse, error) {
	start := time.Now()

	fullPrompt := req.UserPrompt
	if req.SystemPrompt != "" {
		fullPrompt = fmt.Sprintf("[System: %s]\n\n%s", req.SystemPrompt, req.UserPrompt)
	}

	args := []string{"-p", fullPrompt}
	if req.Model != "" {
		args = append(args, "-m", req.Model)
	}

	cmd := exec.CommandContext(ctx, "gemini", args...)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("gemini cli failed: %w (stderr: %s)", err, stderr.String())
	}

	return &CompletionResponse{
		Content: stdout.String(),
		Latency: time.Since(start),
		Tokens:  TokenUsage{},
	}, nil
}

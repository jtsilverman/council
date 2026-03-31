package provider

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"time"
)

// CLIProvider uses `claude --print` to run completions via subscription ($0 cost).
type CLIProvider struct {
	modelOverride string
}

// NewCLIProvider creates a CLI provider. If modelOverride is empty, uses claude's default.
func NewCLIProvider(modelOverride string) *CLIProvider {
	return &CLIProvider{modelOverride: modelOverride}
}

func (p *CLIProvider) Name() string { return "cli" }

func (p *CLIProvider) Complete(ctx context.Context, req CompletionRequest) (*CompletionResponse, error) {
	start := time.Now()

	args := []string{"--print"}
	if req.Model != "" {
		args = append(args, "--model", req.Model)
	} else if p.modelOverride != "" {
		args = append(args, "--model", p.modelOverride)
	}

	// Build the full prompt with system context
	fullPrompt := req.UserPrompt
	if req.SystemPrompt != "" {
		fullPrompt = fmt.Sprintf("[System: %s]\n\n%s", req.SystemPrompt, req.UserPrompt)
	}

	cmd := exec.CommandContext(ctx, "claude", args...)
	cmd.Stdin = bytes.NewBufferString(fullPrompt)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("claude --print failed: %w (stderr: %s)", err, stderr.String())
	}

	return &CompletionResponse{
		Content: stdout.String(),
		Latency: time.Since(start),
		Tokens:  TokenUsage{}, // CLI mode doesn't report tokens
	}, nil
}

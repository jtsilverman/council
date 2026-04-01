package provider

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"time"
)

// CodexProvider uses `codex exec` to run completions via ChatGPT Plus subscription ($0).
type CodexProvider struct {
	model string
}

func NewCodexProvider(model string) *CodexProvider {
	if model == "" {
		model = "gpt-5.4"
	}
	return &CodexProvider{model: model}
}

func (p *CodexProvider) Name() string { return "codex" }

func (p *CodexProvider) Complete(ctx context.Context, req CompletionRequest) (*CompletionResponse, error) {
	start := time.Now()

	model := req.Model
	if model == "" {
		model = p.model
	}

	prompt := req.UserPrompt
	if req.SystemPrompt != "" {
		prompt = fmt.Sprintf("[System: %s]\n\n%s", req.SystemPrompt, req.UserPrompt)
	}

	cmd := exec.CommandContext(ctx, "codex", "exec", prompt)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("codex exec failed: %w (stderr: %s)", err, stderr.String())
	}

	return &CompletionResponse{
		Content: stdout.String(),
		Latency: time.Since(start),
		Tokens:  TokenUsage{},
	}, nil
}

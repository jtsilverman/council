package provider

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"time"
)

// CodexCLIProvider uses `codex exec` to run completions via the OpenAI Codex CLI.
type CodexCLIProvider struct{}

// NewCodexCLIProvider creates a Codex CLI provider.
func NewCodexCLIProvider() *CodexCLIProvider {
	return &CodexCLIProvider{}
}

func (p *CodexCLIProvider) Name() string { return "codex-cli" }

func (p *CodexCLIProvider) Complete(ctx context.Context, req CompletionRequest) (*CompletionResponse, error) {
	start := time.Now()

	fullPrompt := req.UserPrompt
	if req.SystemPrompt != "" {
		fullPrompt = fmt.Sprintf("[System: %s]\n\n%s", req.SystemPrompt, req.UserPrompt)
	}

	args := []string{"exec"}
	if req.Model != "" {
		args = append(args, "--model", req.Model)
	}
	args = append(args, fullPrompt)

	cmd := exec.CommandContext(ctx, "codex", args...)

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

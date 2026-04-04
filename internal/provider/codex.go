package provider

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"
	"unicode/utf8"
)

// CodexFailureKind classifies codex exec outcomes for retry decisions.
type CodexFailureKind string

const (
	CodexOK        CodexFailureKind = "ok"
	CodexExhausted CodexFailureKind = "exhausted"
	CodexTimeout   CodexFailureKind = "timeout"
	CodexEmpty     CodexFailureKind = "empty"
	CodexOther     CodexFailureKind = "other"
)

// CodexProvider uses `codex exec` to run completions via ChatGPT Plus subscription ($0).
type CodexProvider struct {
	model    string
	fallback Provider // optional OpenAI API fallback, set by detect.go
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

	prompt := buildCodexPrompt(req)

	stdout, stderr, err := runCodex(ctx, prompt)
	kind := classifyCodexFailure(err, stdout, stderr)

	if kind == CodexOK {
		return &CompletionResponse{
			Content: stdout,
			Latency: time.Since(start),
			Tokens:  TokenUsage{},
		}, nil
	}

	// CodexOther: no retry, return immediately
	if kind == CodexOther {
		return nil, fmt.Errorf("codex exec failed (%s): %w (stderr: %s)", kind, err, stderr)
	}

	// Retryable failures: CodexExhausted, CodexEmpty, CodexTimeout
	fmt.Fprintf(os.Stderr, "codex: %s, retrying with tighter prompt\n", kind)

	trimmedPrompt := trimPromptForRetry(prompt)
	stdout2, stderr2, err2 := runCodex(ctx, trimmedPrompt)
	kind2 := classifyCodexFailure(err2, stdout2, stderr2)

	if kind2 == CodexOK {
		return &CompletionResponse{
			Content: stdout2,
			Latency: time.Since(start),
			Tokens:  TokenUsage{},
		}, nil
	}

	// Retry failed — try fallback if available
	if p.fallback != nil {
		return p.fallback.Complete(ctx, req)
	}

	if err2 != nil {
		return nil, fmt.Errorf("codex exec retry failed (%s then %s): %w (stderr: %s)", kind, kind2, err2, stderr2)
	}
	return nil, fmt.Errorf("codex exec retry failed (%s then %s): empty output (stderr: %s)", kind, kind2, stderr2)
}

// buildCodexPrompt assembles the prompt string from a CompletionRequest.
func buildCodexPrompt(req CompletionRequest) string {
	prompt := req.UserPrompt
	if req.SystemPrompt != "" {
		prompt = fmt.Sprintf("[System: %s]\n\n%s", req.SystemPrompt, req.UserPrompt)
	}
	return prompt
}

// runCodex executes `codex exec` and returns stdout, stderr, and any error.
func runCodex(ctx context.Context, prompt string) (stdout, stderr string, err error) {
	cmd := exec.CommandContext(ctx, "codex", "exec", prompt)

	var outBuf, errBuf bytes.Buffer
	cmd.Stdout = &outBuf
	cmd.Stderr = &errBuf

	err = cmd.Run()
	return outBuf.String(), errBuf.String(), err
}

// classifyCodexFailure determines the failure kind from codex exec results.
func classifyCodexFailure(err error, stdout, stderr string) CodexFailureKind {
	if err != nil {
		errMsg := err.Error()
		if strings.Contains(errMsg, "context deadline exceeded") || strings.Contains(errMsg, "signal: killed") {
			return CodexTimeout
		}
		stderrLower := strings.ToLower(stderr)
		if strings.Contains(stderrLower, "turn limit") || strings.Contains(stderrLower, "turn count") ||
			strings.Contains(stderrLower, "token limit") || strings.Contains(stderrLower, "token count") ||
			(strings.Contains(stderrLower, "limit") && (strings.Contains(stderrLower, "token") || strings.Contains(stderrLower, "turn"))) {
			return CodexExhausted
		}
		return CodexOther
	}

	if strings.TrimSpace(stdout) == "" {
		return CodexEmpty
	}

	return CodexOK
}

// trimPromptForRetry truncates the prompt to fit within CodexRetryProfile budget.
// It cuts at the last newline before the budget limit to avoid splitting mid-line.
func trimPromptForRetry(prompt string) string {
	budget := CodexRetryProfile().ReviewBudgetChars
	if len(prompt) <= budget {
		return prompt
	}

	// Find last newline before budget (rune-safe)
	truncated := prompt[:budget]
	lastNL := strings.LastIndex(truncated, "\n")
	if lastNL > 0 {
		return truncated[:lastNL]
	}
	// No newline found -- back up to rune boundary
	for budget > 0 && !utf8.RuneStart(prompt[budget]) {
		budget--
	}
	return prompt[:budget]
}

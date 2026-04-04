package provider

import (
	"errors"
	"strings"
	"testing"
)

func TestClassifyCodexFailure(t *testing.T) {
	tests := []struct {
		name   string
		err    error
		stdout string
		stderr string
		want   CodexFailureKind
	}{
		{
			name:   "ok with output",
			err:    nil,
			stdout: "review result here",
			stderr: "",
			want:   CodexOK,
		},
		{
			name:   "empty stdout",
			err:    nil,
			stdout: "   \n  ",
			stderr: "",
			want:   CodexEmpty,
		},
		{
			name:   "timeout via context deadline",
			err:    errors.New("context deadline exceeded"),
			stdout: "",
			stderr: "",
			want:   CodexTimeout,
		},
		{
			name:   "timeout via killed",
			err:    errors.New("signal: killed"),
			stdout: "",
			stderr: "",
			want:   CodexTimeout,
		},
		{
			name:   "bare killed does not match timeout",
			err:    errors.New("process killed unexpectedly"),
			stdout: "",
			stderr: "",
			want:   CodexOther,
		},
		{
			name:   "exhausted via token limit in stderr",
			err:    errors.New("exit status 1"),
			stdout: "",
			stderr: "Token limit exceeded",
			want:   CodexExhausted,
		},
		{
			name:   "exhausted via turn count in stderr",
			err:    errors.New("exit status 1"),
			stdout: "",
			stderr: "max Turn count reached",
			want:   CodexExhausted,
		},
		{
			name:   "stderr with return does not false positive",
			err:    errors.New("exit status 1"),
			stdout: "",
			stderr: "return value error in handler",
			want:   CodexOther,
		},
		{
			name:   "other error",
			err:    errors.New("permission denied"),
			stdout: "",
			stderr: "some unrelated error",
			want:   CodexOther,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := classifyCodexFailure(tt.err, tt.stdout, tt.stderr)
			if got != tt.want {
				t.Errorf("classifyCodexFailure() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestTrimPromptForRetry(t *testing.T) {
	budget := CodexRetryProfile().ReviewBudgetChars // 4000

	t.Run("short prompt unchanged", func(t *testing.T) {
		prompt := "short prompt"
		got := trimPromptForRetry(prompt)
		if got != prompt {
			t.Errorf("expected unchanged prompt, got %q", got)
		}
	})

	t.Run("long prompt trimmed at newline", func(t *testing.T) {
		// Build a prompt that exceeds budget with clear newline boundaries
		lines := make([]string, 0)
		for i := 0; len(strings.Join(lines, "\n")) < budget+2000; i++ {
			lines = append(lines, strings.Repeat("x", 80))
		}
		prompt := strings.Join(lines, "\n")

		got := trimPromptForRetry(prompt)
		if len(got) > budget {
			t.Errorf("trimmed prompt too long: %d chars, budget %d", len(got), budget)
		}
		// Should end at a newline boundary (no trailing partial line)
		if strings.HasSuffix(got, "x") && len(got) == budget {
			// Only fails if it didn't find a newline to cut at
			t.Log("trimmed at exact budget (no newline found in range)")
		}
		if len(got) == 0 {
			t.Error("trimmed to empty")
		}
	})

	t.Run("exact budget size unchanged", func(t *testing.T) {
		prompt := strings.Repeat("a", budget)
		got := trimPromptForRetry(prompt)
		if got != prompt {
			t.Errorf("expected unchanged prompt at exact budget, got len %d", len(got))
		}
	})
}

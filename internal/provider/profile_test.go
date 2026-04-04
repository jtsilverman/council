package provider

import (
	"context"
	"testing"
)

// fakeProvider is a minimal Provider implementation for testing.
type fakeProvider struct {
	name string
}

func (f *fakeProvider) Complete(_ context.Context, _ CompletionRequest) (*CompletionResponse, error) {
	return &CompletionResponse{}, nil
}

func (f *fakeProvider) Name() string { return f.name }

func TestProfileDefaults(t *testing.T) {
	p := DefaultProfile()
	if p.ReviewBudgetChars != 100_000 {
		t.Errorf("ReviewBudgetChars = %d, want 100000", p.ReviewBudgetChars)
	}
	if p.DebateBudgetChars != 100_000 {
		t.Errorf("DebateBudgetChars = %d, want 100000", p.DebateBudgetChars)
	}
	if p.SynthesisBudgetChars != 100_000 {
		t.Errorf("SynthesisBudgetChars = %d, want 100000", p.SynthesisBudgetChars)
	}
	if p.RequireStructuredDigest {
		t.Error("RequireStructuredDigest should be false")
	}
	if p.PreferWorkspaceRead {
		t.Error("PreferWorkspaceRead should be false")
	}
}

func TestProfileCodex(t *testing.T) {
	p := CodexProfile()
	if p.ReviewBudgetChars != 8_000 {
		t.Errorf("ReviewBudgetChars = %d, want 8000", p.ReviewBudgetChars)
	}
	if p.DebateBudgetChars != 6_000 {
		t.Errorf("DebateBudgetChars = %d, want 6000", p.DebateBudgetChars)
	}
	if p.SynthesisBudgetChars != 6_000 {
		t.Errorf("SynthesisBudgetChars = %d, want 6000", p.SynthesisBudgetChars)
	}
	if !p.RequireStructuredDigest {
		t.Error("RequireStructuredDigest should be true")
	}
	if !p.PreferWorkspaceRead {
		t.Error("PreferWorkspaceRead should be true")
	}
}

func TestProfileCodexRetry(t *testing.T) {
	p := CodexRetryProfile()
	if p.ReviewBudgetChars != 4_000 {
		t.Errorf("ReviewBudgetChars = %d, want 4000", p.ReviewBudgetChars)
	}
	if p.DebateBudgetChars != 3_000 {
		t.Errorf("DebateBudgetChars = %d, want 3000", p.DebateBudgetChars)
	}
	if p.SynthesisBudgetChars != 3_000 {
		t.Errorf("SynthesisBudgetChars = %d, want 3000", p.SynthesisBudgetChars)
	}
	if !p.RequireStructuredDigest {
		t.Error("RequireStructuredDigest should be true")
	}
	if !p.PreferWorkspaceRead {
		t.Error("PreferWorkspaceRead should be true")
	}
}

func TestProfileFor(t *testing.T) {
	tests := []struct {
		name     string
		provider Provider
		wantFn   func() PromptProfile
	}{
		{"codex", &fakeProvider{name: "codex"}, CodexProfile},
		{"claude", &fakeProvider{name: "claude"}, DefaultProfile},
		{"gemini", &fakeProvider{name: "gemini"}, DefaultProfile},
		{"openai", &fakeProvider{name: "openai"}, DefaultProfile},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ProfileFor(tt.provider)
			want := tt.wantFn()
			if got != want {
				t.Errorf("ProfileFor(%q) = %+v, want %+v", tt.name, got, want)
			}
		})
	}
}

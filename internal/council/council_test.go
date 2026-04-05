package council

import (
	"context"
	"testing"
	"time"

	"github.com/jtsilverman/council/internal/provider"
)

// mockProvider implements provider.Provider for testing.
type mockProvider struct {
	name string
}

func (m *mockProvider) Complete(_ context.Context, _ provider.CompletionRequest) (*provider.CompletionResponse, error) {
	return &provider.CompletionResponse{Content: "mock"}, nil
}

func (m *mockProvider) Name() string { return m.name }

func TestProviders_For_Default(t *testing.T) {
	p := &Providers{
		Default: &mockProvider{name: "default"},
	}

	got := p.For(0)
	if got.Name() != "default" {
		t.Errorf("For(0) = %q, want default", got.Name())
	}
	got = p.For(99)
	if got.Name() != "default" {
		t.Errorf("For(99) = %q, want default", got.Name())
	}
}

func TestProviders_For_PerModel(t *testing.T) {
	p := &Providers{
		Default: &mockProvider{name: "default"},
		PerModel: map[int]provider.Provider{
			1: &mockProvider{name: "special"},
		},
	}

	got := p.For(0)
	if got.Name() != "default" {
		t.Errorf("For(0) = %q, want default", got.Name())
	}

	got = p.For(1)
	if got.Name() != "special" {
		t.Errorf("For(1) = %q, want special", got.Name())
	}

	got = p.For(2)
	if got.Name() != "default" {
		t.Errorf("For(2) = %q, want default (fallback)", got.Name())
	}
}

func TestProviders_For_NilPerModel(t *testing.T) {
	p := &Providers{
		Default:  &mockProvider{name: "default"},
		PerModel: nil,
	}

	got := p.For(0)
	if got.Name() != "default" {
		t.Errorf("For(0) with nil PerModel = %q, want default", got.Name())
	}
}

func TestTokenAggregation(t *testing.T) {
	c := &Council{
		Name:     "test",
		Strategy: "mock",
	}

	strat := &mockStrategy{
		result: &Deliberation{
			Rounds: []Round{
				{
					Phase: "review",
					Responses: []Response{
						{Member: "A", Tokens: TokenUsage{Input: 100, Output: 50, Cost: 0.01}},
						{Member: "B", Tokens: TokenUsage{Input: 200, Output: 80, Cost: 0.02}},
					},
				},
				{
					Phase: "debate",
					Responses: []Response{
						{Member: "A", Tokens: TokenUsage{Input: 150, Output: 60, Cost: 0.015}},
					},
				},
			},
			Synthesis: Response{
				Member: "Chair",
				Tokens: TokenUsage{Input: 300, Output: 100, Cost: 0.03},
			},
		},
	}

	p := &Providers{Default: &mockProvider{name: "test"}}
	delib, err := Run(context.Background(), c, "test query", p, strat)
	if err != nil {
		t.Fatalf("Run() error: %v", err)
	}

	// Input: 100 + 200 + 150 + 300 = 750
	if delib.TotalTokens.Input != 750 {
		t.Errorf("TotalTokens.Input = %d, want 750", delib.TotalTokens.Input)
	}
	// Output: 50 + 80 + 60 + 100 = 290
	if delib.TotalTokens.Output != 290 {
		t.Errorf("TotalTokens.Output = %d, want 290", delib.TotalTokens.Output)
	}
	// Cost: 0.01 + 0.02 + 0.015 + 0.03 = 0.075
	wantCost := 0.075
	if delib.TotalCost < wantCost-0.001 || delib.TotalCost > wantCost+0.001 {
		t.Errorf("TotalCost = %f, want ~%f", delib.TotalCost, wantCost)
	}

	if delib.Query != "test query" {
		t.Errorf("Query = %q", delib.Query)
	}
	if delib.Council != "test" {
		t.Errorf("Council = %q", delib.Council)
	}
	if delib.Strategy != "mock" {
		t.Errorf("Strategy = %q", delib.Strategy)
	}
	if delib.Duration <= 0 {
		t.Error("Duration should be positive")
	}
}

// mockStrategy implements Strategy for testing.
type mockStrategy struct {
	result *Deliberation
}

func (m *mockStrategy) Run(_ context.Context, _ *Council, _ string, _ *Providers) (*Deliberation, error) {
	time.Sleep(time.Millisecond)
	return m.result, nil
}

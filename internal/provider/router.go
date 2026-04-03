package provider

import (
	"context"
	"fmt"
	"strings"
	"sync"
)

// Router implements Provider by dispatching each request to the correct
// underlying provider based on the model name or an explicit provider hint.
//
// Model prefix mapping (CLI mode):
//
//	claude-*, sonnet-*, haiku-*, opus-*  → claude --print
//	gpt-*, o1*, o3*, o4*                → codex exec
//	gemini-*                             → gemini CLI
//
// When --api is used, the same routing applies but via API providers.
type Router struct {
	useAPI   bool
	fallback Provider // default when model prefix is unrecognized

	mu        sync.Mutex
	providers map[string]Provider // cache by provider name
}

// NewRouter creates a provider router.
// If useAPI is true, non-Claude models use their respective API providers.
// If useAPI is false, non-Claude models use their respective CLI wrappers.
func NewRouter(useAPI bool, fallback Provider) *Router {
	return &Router{
		useAPI:    useAPI,
		fallback:  fallback,
		providers: make(map[string]Provider),
	}
}

func (r *Router) Name() string { return "router" }

func (r *Router) Complete(ctx context.Context, req CompletionRequest) (*CompletionResponse, error) {
	p, err := r.resolve(req.Model)
	if err != nil {
		return nil, err
	}
	return p.Complete(ctx, req)
}

// resolve picks the right provider for a given model string.
func (r *Router) resolve(model string) (Provider, error) {
	name := providerForModel(model)
	if name == "" {
		return r.fallback, nil
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	if p, ok := r.providers[name]; ok {
		return p, nil
	}

	p, err := r.create(name)
	if err != nil {
		return nil, fmt.Errorf("create %s provider for model %q: %w", name, model, err)
	}
	r.providers[name] = p
	return p, nil
}

func (r *Router) create(name string) (Provider, error) {
	if r.useAPI {
		return r.createAPI(name)
	}
	return r.createCLI(name)
}

func (r *Router) createCLI(name string) (Provider, error) {
	switch name {
	case "claude":
		return NewCLIProvider(""), nil
	case "codex":
		return NewCodexCLIProvider(), nil
	case "gemini":
		return NewGeminiCLIProvider(), nil
	default:
		return r.fallback, nil
	}
}

func (r *Router) createAPI(name string) (Provider, error) {
	switch name {
	case "claude":
		return NewAnthropicProvider()
	case "codex":
		return NewOpenAIProvider()
	case "gemini":
		return NewGeminiProvider()
	default:
		return r.fallback, nil
	}
}

// providerForModel returns the provider name for a model string.
// Returns "" if the model should use the fallback provider.
func providerForModel(model string) string {
	m := strings.ToLower(model)
	switch {
	case strings.HasPrefix(m, "claude-"),
		strings.HasPrefix(m, "sonnet-"),
		strings.HasPrefix(m, "haiku-"),
		strings.HasPrefix(m, "opus-"):
		return "claude"
	case strings.HasPrefix(m, "gpt-"),
		strings.HasPrefix(m, "o1"),
		strings.HasPrefix(m, "o3"),
		strings.HasPrefix(m, "o4"):
		return "codex"
	case strings.HasPrefix(m, "gemini-"):
		return "gemini"
	default:
		return ""
	}
}

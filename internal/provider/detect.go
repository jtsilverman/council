package provider

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// DetectProvider auto-detects the best provider for a model name.
// Priority: CLI subscription (free) > API key > OpenRouter fallback.
func DetectProvider(modelName string) (Provider, error) {
	// Claude models → claude --print (free) or Anthropic API
	if isClaudeModel(modelName) {
		if cliAvailable("claude") {
			return NewCLIProvider(modelName), nil
		}
		if hasEnv("ANTHROPIC_API_KEY") {
			return NewAnthropicProvider()
		}
		return nil, fmt.Errorf("claude model %q requested but no claude CLI or ANTHROPIC_API_KEY found", modelName)
	}

	// GPT/o-series models → codex exec (free) or OpenAI API
	if isOpenAIModel(modelName) {
		if cliAvailable("codex") {
			cp := NewCodexProvider(modelName)
			if hasEnv("OPENAI_API_KEY") {
				if oai, err := NewOpenAIProvider(); err == nil {
					cp.fallback = oai
				}
			}
			return cp, nil
		}
		if hasEnv("OPENAI_API_KEY") {
			return NewOpenAIProvider()
		}
		return nil, fmt.Errorf("openai model %q requested but no codex CLI or OPENAI_API_KEY found", modelName)
	}

	// Gemini models → gemini CLI (free) or Gemini API
	if isGeminiModel(modelName) {
		if cliAvailable("gemini") {
			return NewGeminiCLIProvider(modelName), nil
		}
		if hasEnv("GEMINI_API_KEY") {
			return NewGeminiProvider()
		}
		return nil, fmt.Errorf("gemini model %q requested but no gemini CLI or GEMINI_API_KEY found", modelName)
	}

	// Ollama models (prefix ollama:)
	if isOllamaModel(modelName) {
		ollamaModel := strings.TrimPrefix(modelName, "ollama:")
		p := NewOllamaProvider("")
		return &ollamaModelOverride{p, ollamaModel}, nil
	}

	// Kimi models → Kimi API (OpenAI-compatible)
	if isKimiModel(modelName) {
		if hasEnv("KIMI_API_KEY") {
			return NewKimiProvider()
		}
		return nil, fmt.Errorf("kimi model %q requested but KIMI_API_KEY not set", modelName)
	}

	// MiniMax models → MiniMax API (OpenAI-compatible)
	if isMiniMaxModel(modelName) {
		if hasEnv("MINIMAX_API_KEY") {
			return NewMiniMaxProvider()
		}
		return nil, fmt.Errorf("minimax model %q requested but MINIMAX_API_KEY not set", modelName)
	}

	// OpenRouter fallback (any model)
	if hasEnv("OPENROUTER_API_KEY") {
		return NewOpenRouterProvider()
	}

	return nil, fmt.Errorf("no provider for model %q. Run 'council providers' to see what's available", modelName)
}

// DetectDefault returns the default provider (Claude CLI if available).
func DetectDefault() Provider {
	if cliAvailable("claude") {
		return NewCLIProvider("")
	}
	p, err := NewAnthropicProvider()
	if err == nil {
		return p
	}
	// Last resort: CLI provider (will fail at call time if claude not found)
	return NewCLIProvider("")
}

// ProviderStatus represents the availability of a provider.
type ProviderStatus struct {
	Name      string
	Available bool
	Method    string // "cli", "api", "local"
	Detail    string
	Models    []string
}

// DetectAll checks all available providers and returns their status.
func DetectAll() []ProviderStatus {
	var statuses []ProviderStatus

	// Claude
	if cliAvailable("claude") {
		statuses = append(statuses, ProviderStatus{"Claude", true, "cli", "claude --print (subscription, $0)", []string{"claude-opus-4-6", "claude-sonnet-4-6", "claude-haiku-4-5"}})
	} else if hasEnv("ANTHROPIC_API_KEY") {
		statuses = append(statuses, ProviderStatus{"Claude", true, "api", "ANTHROPIC_API_KEY", []string{"claude-opus-4-6", "claude-sonnet-4-6", "claude-haiku-4-5"}})
	} else {
		statuses = append(statuses, ProviderStatus{"Claude", false, "", "install Claude Code or set ANTHROPIC_API_KEY", nil})
	}

	// GPT/OpenAI
	if cliAvailable("codex") {
		statuses = append(statuses, ProviderStatus{"OpenAI", true, "cli", "codex exec (ChatGPT Plus, $0)", []string{"gpt-5.4", "gpt-5.4-mini", "gpt-5.4-nano", "o3", "o4-mini"}})
	} else if hasEnv("OPENAI_API_KEY") {
		statuses = append(statuses, ProviderStatus{"OpenAI", true, "api", "OPENAI_API_KEY", []string{"gpt-5.4", "gpt-5.4-mini", "gpt-5.4-nano", "o3", "o4-mini"}})
	} else {
		statuses = append(statuses, ProviderStatus{"OpenAI", false, "", "install Codex CLI or set OPENAI_API_KEY", nil})
	}

	// Gemini
	if cliAvailable("gemini") {
		statuses = append(statuses, ProviderStatus{"Gemini", true, "cli", "gemini CLI (Google subscription, $0)", []string{"gemini-2.5-pro", "gemini-2.5-flash", "gemini-3.1-pro-preview"}})
	} else if hasEnv("GEMINI_API_KEY") {
		statuses = append(statuses, ProviderStatus{"Gemini", true, "api", "GEMINI_API_KEY", []string{"gemini-2.5-pro", "gemini-2.5-flash", "gemini-3.1-pro-preview"}})
	} else {
		statuses = append(statuses, ProviderStatus{"Gemini", false, "", "install Gemini CLI or set GEMINI_API_KEY", nil})
	}

	// Ollama
	statuses = append(statuses, ProviderStatus{"Ollama", true, "local", "localhost:11434 (prefix with ollama:)", []string{"ollama:llama3.2", "ollama:qwen2.5", "ollama:deepseek-v3", "ollama:mistral-large"}})

	// Kimi
	if hasEnv("KIMI_API_KEY") {
		statuses = append(statuses, ProviderStatus{"Kimi", true, "api", "KIMI_API_KEY", []string{"kimi-k2.5", "kimi-k2-thinking"}})
	} else {
		statuses = append(statuses, ProviderStatus{"Kimi", false, "", "set KIMI_API_KEY", nil})
	}

	// MiniMax
	if hasEnv("MINIMAX_API_KEY") {
		statuses = append(statuses, ProviderStatus{"MiniMax", true, "api", "MINIMAX_API_KEY", []string{"MiniMax-M2.7", "MiniMax-M2.7-highspeed"}})
	} else {
		statuses = append(statuses, ProviderStatus{"MiniMax", false, "", "set MINIMAX_API_KEY", nil})
	}

	// OpenRouter
	if hasEnv("OPENROUTER_API_KEY") {
		statuses = append(statuses, ProviderStatus{"OpenRouter", true, "api", "OPENROUTER_API_KEY (any model)", []string{"any model via openrouter.ai"}})
	} else {
		statuses = append(statuses, ProviderStatus{"OpenRouter", false, "", "set OPENROUTER_API_KEY (fallback for any model)", nil})
	}

	return statuses
}

func isClaudeModel(m string) bool {
	m = strings.ToLower(m)
	return strings.HasPrefix(m, "claude") || m == "opus" || m == "sonnet" || m == "haiku"
}

func isOpenAIModel(m string) bool {
	m = strings.ToLower(m)
	return strings.HasPrefix(m, "gpt") || strings.HasPrefix(m, "o1") || strings.HasPrefix(m, "o3") || strings.HasPrefix(m, "o4")
}

func isGeminiModel(m string) bool {
	return strings.HasPrefix(strings.ToLower(m), "gemini")
}

func isOllamaModel(m string) bool {
	return strings.HasPrefix(strings.ToLower(m), "ollama:")
}

func isKimiModel(m string) bool {
	return strings.HasPrefix(strings.ToLower(m), "kimi")
}

func isMiniMaxModel(m string) bool {
	return strings.HasPrefix(strings.ToLower(m), "minimax")
}

func cliAvailable(name string) bool {
	_, err := exec.LookPath(name)
	return err == nil
}

func hasEnv(key string) bool {
	return os.Getenv(key) != ""
}

// ollamaModelOverride wraps OllamaProvider to override the model name.
type ollamaModelOverride struct {
	*OllamaProvider
	model string
}

func (o *ollamaModelOverride) Complete(ctx context.Context, req CompletionRequest) (*CompletionResponse, error) {
	req.Model = o.model
	return o.OllamaProvider.Complete(ctx, req)
}

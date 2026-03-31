# Council

## Overview

A Go CLI for multi-perspective LLM deliberation. Send any query through a "council" of LLM reviewers, each with a different expert persona, then run a debate round where they challenge each other's findings, and a chair synthesizes the final answer. The twist: designed diversity via persona prompts beats random diversity via different models. Unlike Karpathy's llm-council (Python web app, 16K stars, "vibe coded"), this is a production CLI that pipes into any workflow, tracks cost/latency, and ships with configurable deliberation strategies. No Go-based council tool exists.

## Scope

### Phase 1 — Core: Single-Provider Council

- Go CLI: `council "query"` runs a 3-member council and outputs the synthesized result
- Three deliberation phases: parallel review → debate → chair synthesis
- Built-in councils: `general` (default), `code-review`, `writing`
- Each council defines 3-5 expert personas with distinct system prompts
- CLI mode is the default: spawns parallel `claude --print` processes ($0, uses subscription)
- API mode opt-in: `--api` flag to use Anthropic API instead (faster, costs money)
- Streaming output: show each reviewer's response as it arrives
- Cost and latency tracking per council run
- Pipe-friendly: `echo "query" | council` and `cat file.py | council --council code-review`
- JSON output mode for automation
- Configurable council size (2-7 members)

### Phase 2 — Full Product: Multi-Provider + Custom Councils

- Multi-provider support: OpenAI, Gemini, Ollama (local models)
- OpenRouter support (single API key for all cloud models)
- Mixed councils: different members can use different providers/models
- Custom council definitions via YAML config file
- Debate strategies: `debate` (default, multi-round), `vote` (majority wins), `ranked` (pairwise comparison)
- `--strategy` flag to pick deliberation method
- Per-member model override (e.g., chair always uses the strongest model)
- `council list` to show available councils and their members
- `council explain` to show the deliberation trace (who said what, who challenged whom)
- Probe integration: `probe --council` flag that uses council for code review
- Ship: README, `go install`, goreleaser binaries

### Phase 3 — Stretch

- ELO-style model ranking based on debate outcomes (which model's findings survive most often)
- Council replay: save deliberation traces and replay them
- Web UI: real-time visualization of the deliberation process
- MCP server mode: expose council as a tool for other AI agents

### Not building (any phase)

- User accounts or cloud hosting
- Training or fine-tuning models
- RAG or document retrieval (council reviews what you give it)
- GUI or desktop app

### Ship target

`go install`, GitHub releases via goreleaser, brew tap.

## Project Type

Pure code (CLI tool with LLM API integrations).

## Stack

- **Language:** Go 1.24
- **LLM SDKs:** anthropic-sdk-go (Anthropic), go-openai (OpenAI), custom HTTP for Gemini/Ollama/OpenRouter
- **CLI:** cobra (same as Probe, consistent)
- **Config:** yaml.v3
- **Concurrency:** stdlib goroutines + channels (parallel reviewer execution)
- **Output:** lipgloss for terminal styling
- **Why Go:** Fast CLI, excellent concurrency for parallel API calls, matches Probe for integration, adds another serious Go project to portfolio.

## Architecture

```
council/
├── main.go
├── cmd/
│   ├── root.go                # Cobra root, global flags, config loading
│   ├── run.go                 # Default command: run a council deliberation
│   ├── list.go                # List available councils
│   └── explain.go             # Show deliberation trace
├── internal/
│   ├── council/
│   │   ├── council.go         # Council orchestrator (parallel → debate → synthesize)
│   │   ├── member.go          # Council member (persona, provider, model)
│   │   └── deliberation.go    # Deliberation trace (who said what)
│   ├── provider/
│   │   ├── provider.go        # Provider interface
│   │   ├── anthropic.go       # Anthropic (Claude) provider
│   │   ├── openai.go          # OpenAI provider
│   │   ├── gemini.go          # Gemini provider
│   │   ├── ollama.go          # Ollama (local) provider
│   │   ├── openrouter.go      # OpenRouter provider
│   │   └── cli.go             # Claude CLI provider (--cli mode)
│   ├── strategy/
│   │   ├── strategy.go        # Strategy interface
│   │   ├── debate.go          # Multi-round debate strategy
│   │   ├── vote.go            # Majority vote strategy
│   │   └── ranked.go          # Pairwise ranking strategy
│   ├── persona/
│   │   ├── registry.go        # Built-in council definitions
│   │   ├── general.go         # General-purpose council personas
│   │   ├── codereview.go      # Code review council personas
│   │   └── writing.go         # Writing council personas
│   ├── config/
│   │   └── config.go          # YAML config loading, custom councils
│   └── output/
│       ├── terminal.go        # Colored terminal output with streaming
│       ├── json.go            # JSON output
│       └── trace.go           # Deliberation trace renderer
├── councils/
│   └── example.yaml           # Example custom council definition
├── tests/
│   ├── council_test.go
│   └── strategy_test.go
├── go.mod
├── go.sum
├── README.md
├── .goreleaser.yml
└── .github/
    └── workflows/
        └── release.yml
```

### Data Models

```go
// Member represents one council member with a specific expert persona.
type Member struct {
    Name     string   // "Security Auditor", "Performance Engineer"
    Persona  string   // System prompt defining their expertise and perspective
    Provider string   // "anthropic", "openai", "gemini", "ollama", "openrouter", "cli"
    Model    string   // "claude-sonnet-4-20250514", "gpt-4o", etc.
}

// Council is a named group of members with a deliberation strategy.
type Council struct {
    Name        string    // "code-review", "general", "writing"
    Description string
    Members     []Member
    Chair       Member    // Synthesizes final result (typically strongest model)
    Strategy    string    // "debate", "vote", "ranked"
    MaxRounds   int       // For debate strategy: max debate rounds (default 1)
}

// Response is a single member's response to a query or debate prompt.
type Response struct {
    Member    string        // Member name
    Content   string        // Full response text
    Tokens    TokenUsage
    Latency   time.Duration
}

// TokenUsage tracks API usage.
type TokenUsage struct {
    Input  int
    Output int
    Cost   float64 // USD
}

// Deliberation is the full trace of a council run.
type Deliberation struct {
    Query       string
    Council     string
    Strategy    string
    Rounds      []Round       // Each round of deliberation
    Synthesis   Response      // Chair's final synthesis
    TotalTokens TokenUsage
    TotalCost   float64
    Duration    time.Duration
}

// Round represents one phase of deliberation.
type Round struct {
    Phase     string     // "review", "debate", "synthesis"
    Responses []Response
}
```

### Provider Interface

```go
type Provider interface {
    // Complete sends a prompt and returns the response.
    Complete(ctx context.Context, req CompletionRequest) (*CompletionResponse, error)
    // Name returns the provider identifier.
    Name() string
}

type CompletionRequest struct {
    SystemPrompt string
    UserPrompt   string
    Model        string
    MaxTokens    int
}

type CompletionResponse struct {
    Content string
    Tokens  TokenUsage
}
```

### Strategy Interface

```go
type Strategy interface {
    // Run executes the deliberation strategy and returns the final synthesis.
    Run(ctx context.Context, council *Council, query string, providers map[string]Provider) (*Deliberation, error)
}
```

### Built-in Council: Code Review

```go
var CodeReviewCouncil = Council{
    Name: "code-review",
    Description: "Multi-perspective code review council",
    Members: []Member{
        {
            Name:    "Security Auditor",
            Persona: "You are a penetration tester and application security expert. Your job is to find every way this code could be exploited: injection, auth bypass, data exposure, insecure defaults, missing validation. You think like an attacker.",
        },
        {
            Name:    "Performance Engineer",
            Persona: "You are obsessed with efficiency. Find bottlenecks, unnecessary allocations, O(n^2) loops hidden behind clean abstractions, N+1 queries, missing caching opportunities, memory leaks. You think in terms of flame graphs and benchmarks.",
        },
        {
            Name:    "Bug Hunter",
            Persona: "You find logic errors that pass tests. Off-by-one errors, race conditions, nil dereferences, unchecked error returns, edge cases at boundaries, incorrect assumptions about input. You think adversarially about what inputs could break this.",
        },
        {
            Name:    "Maintainability Critic",
            Persona: "You maintain a 10 million line codebase. You care about: will this be readable in 6 months? Are abstractions earning their complexity? Are names precise? Is the API surface minimal? Could a junior dev safely modify this? You think about the next person to touch this code.",
        },
    },
    Chair: Member{
        Name:    "Tech Lead",
        Persona: "You are the technical lead synthesizing a code review. You have seen each reviewer's findings and the debate. Produce a final review that: keeps findings with consensus support, downgrades contested findings to suggestions, drops findings that were successfully challenged. Be decisive. Prioritize by impact. Output a clear, actionable review.",
    },
    Strategy:  "debate",
    MaxRounds: 1,
}
```

### Deliberation Flow (Debate Strategy)

```
Phase 1: Independent Review
  ├── Security Auditor    ──┐
  ├── Performance Engineer ──┤── parallel, each sees query only
  ├── Bug Hunter          ──┤
  └── Maintainability     ──┘
                              ↓
Phase 2: Debate
  Each member sees ALL other members' findings.
  Prompt: "Here are the findings from the other reviewers.
           Challenge any finding you disagree with (explain why).
           Support any finding you think is especially important.
           Add anything the others missed."
  ├── Security Auditor    ──┐
  ├── Performance Engineer ──┤── parallel, each sees all Phase 1 output
  ├── Bug Hunter          ──┤
  └── Maintainability     ──┘
                              ↓
Phase 3: Chair Synthesis
  Chair sees Phase 1 + Phase 2.
  Produces final consolidated response.
  └── Tech Lead ── sees everything, produces final output
```

### CLI Interface

```bash
# Basic usage
council "Should I use a mutex or channel for this Go pattern?"

# Pipe a file for code review
cat main.go | council --council code-review

# Use a specific strategy
council --strategy vote "What's the best database for this use case?"

# JSON output for automation
council --json "Review this architecture decision"

# Show the full deliberation trace
council explain "What's wrong with this code?" < buggy.py

# List available councils
council list

# Use CLI mode (no API key needed)
council --cli "Explain this error message"

# Custom council size
council --members 5 "Complex architectural question"

# Specific provider
council --provider openai --model gpt-4o "Question"
```

### Custom Council YAML

```yaml
name: architecture-review
description: Review system architecture decisions
strategy: debate
max_rounds: 2
members:
  - name: Distributed Systems Expert
    persona: "You specialize in distributed systems..."
    provider: anthropic
    model: claude-sonnet-4-20250514
  - name: Cost Optimizer
    persona: "You think about AWS bills..."
    provider: anthropic
    model: claude-sonnet-4-20250514
  - name: Operations Engineer
    persona: "You think about what breaks at 3 AM..."
    provider: anthropic
    model: claude-sonnet-4-20250514
chair:
  name: VP of Engineering
  persona: "You weigh trade-offs and make final architecture calls..."
  provider: anthropic
  model: claude-sonnet-4-20250514
```

## Task List

## Phase 1: Core

### 1A: Project Foundation

#### Task 1A.1: Initialize Go module and project structure
**Files:** `go.mod` (create), `main.go` (create), `cmd/root.go` (create), `cmd/run.go` (create stub)
**Do:** Initialize Go module as `github.com/jtsilverman/council`. Create directory structure. Set up cobra CLI with root command (global flags: --json, --cli, --provider, --model, --council, --members, --strategy, --verbose). Stub out `run` as the default command. Add global flags: json output, verbose mode.
**Validate:** `cd /Users/rock/Rock/projects/council && export PATH="$HOME/go-sdk/go/bin:$PATH" && go build -o council . && ./council --help`

#### Task 1A.2: Provider interface and Anthropic provider
**Files:** `internal/provider/provider.go` (create), `internal/provider/anthropic.go` (create), `internal/provider/cli.go` (create)
**Do:** Define `Provider` interface with `Complete(ctx, req) (*resp, error)` and `Name() string`. Implement Anthropic provider using anthropic-sdk-go. Implement CLI provider that shells out to `claude --print --model <model>`. Both read API key from env var. Add `NewProvider(name, apiKey string) (Provider, error)` factory function.
**Validate:** `cd /Users/rock/Rock/projects/council && export PATH="$HOME/go-sdk/go/bin:$PATH" && go build -o council . && echo "Say hello in exactly 3 words" | ANTHROPIC_API_KEY=$ANTHROPIC_API_KEY ./council --provider anthropic --model claude-sonnet-4-20250514 2>/dev/null; echo "exit: $?"`

### 1B: Council Engine

#### Task 1B.1: Council and Member data models
**Files:** `internal/council/council.go` (create), `internal/council/member.go` (create), `internal/council/deliberation.go` (create)
**Do:** Define Council, Member, Response, TokenUsage, Deliberation, and Round structs as specified in the architecture section. Add `NewCouncil(name string) (*Council, error)` that looks up built-in councils. Add methods: `council.Run(ctx, query, providers) (*Deliberation, error)` that delegates to the configured strategy.
**Validate:** `cd /Users/rock/Rock/projects/council && export PATH="$HOME/go-sdk/go/bin:$PATH" && go build ./...`

#### Task 1B.2: Debate strategy
**Files:** `internal/strategy/strategy.go` (create), `internal/strategy/debate.go` (create)
**Do:** Define `Strategy` interface. Implement `DebateStrategy` with three phases: (1) Parallel independent review: each member gets the query + their persona as system prompt, run concurrently via goroutines. (2) Debate: each member sees all Phase 1 responses, prompted to challenge/support/add. Run concurrently. (3) Chair synthesis: chair sees Phase 1 + Phase 2, produces final answer. Track tokens and latency per response. Return a Deliberation with all rounds.
**Validate:** `cd /Users/rock/Rock/projects/council && export PATH="$HOME/go-sdk/go/bin:$PATH" && go test ./internal/strategy/ -v`

### 1C: Built-in Councils

#### Task 1C.1: General and code-review councils
**Files:** `internal/persona/registry.go` (create), `internal/persona/general.go` (create), `internal/persona/codereview.go` (create)
**Do:** Create a registry that maps council names to Council definitions. Implement `general` council (3 members: Analytical Thinker, Creative Problem Solver, Practical Engineer + Moderator chair). Implement `code-review` council (4 members: Security Auditor, Performance Engineer, Bug Hunter, Maintainability Critic + Tech Lead chair) as specified in architecture. Each persona has a carefully crafted system prompt. Default provider/model is anthropic/claude-sonnet-4-20250514 for all members, chair uses claude-sonnet-4-20250514.
**Validate:** `cd /Users/rock/Rock/projects/council && export PATH="$HOME/go-sdk/go/bin:$PATH" && go build ./... && echo "ok"`

#### Task 1C.2: Writing council
**Files:** `internal/persona/writing.go` (create)
**Do:** Implement `writing` council (3 members: Editor focused on clarity and structure, Fact Checker focused on accuracy and claims, Audience Advocate focused on reader experience + Executive Editor chair). Register in registry.
**Validate:** `cd /Users/rock/Rock/projects/council && export PATH="$HOME/go-sdk/go/bin:$PATH" && go build ./... && echo "ok"`

### 1D: Output and CLI Wiring

#### Task 1D.1: Terminal and JSON output
**Files:** `internal/output/terminal.go` (create), `internal/output/json.go` (create), `internal/output/trace.go` (create)
**Do:** Terminal output: show each member's name + response during Phase 1 (with color-coded headers using lipgloss), then debate highlights, then the final synthesis prominently. Show cost/latency summary at the end. JSON output: serialize full Deliberation struct. Trace output (for `explain` command): show the complete deliberation with phase headers, member names, and full responses.
**Validate:** `cd /Users/rock/Rock/projects/council && export PATH="$HOME/go-sdk/go/bin:$PATH" && go build ./... && echo "ok"`

#### Task 1D.2: Wire up the run command
**Files:** `cmd/run.go` (modify), `cmd/root.go` (modify)
**Do:** Wire the run command: read query from args or stdin, look up council by name (default: general), create providers for each member, run the council's deliberation strategy, output result via terminal or JSON formatter. Handle --cli flag (use CLI provider), --provider/--model flags (override all members), --members flag (adjust council size), --council flag (select built-in council). Add `council list` subcommand that prints available councils and their members.
**Validate:** `cd /Users/rock/Rock/projects/council && export PATH="$HOME/go-sdk/go/bin:$PATH" && go build -o council . && echo "What are the trade-offs of Go vs Rust for CLI tools? Answer in 2 sentences each." | ./council --cli 2>&1 | head -50`

### 1E: End-to-End Test

#### Task 1E.1: Integration test
**Files:** `tests/council_test.go` (create)
**Do:** Test the full pipeline with the CLI provider (no API key needed): (1) general council with a simple question, verify Deliberation has 3 rounds (review, debate, synthesis), (2) code-review council with a code snippet, verify response mentions security or bugs, (3) JSON output parses correctly, (4) list command shows all 3 built-in councils. Use `--cli` mode for all tests to avoid API costs.
**Validate:** `cd /Users/rock/Rock/projects/council && export PATH="$HOME/go-sdk/go/bin:$PATH" && go test ./tests/ -v -timeout 120s`

## Phase 2: Full Product

### 2A: Multi-Provider Support

#### Task 2A.1: OpenAI provider
**Files:** `internal/provider/openai.go` (create)
**Do:** Implement OpenAI provider using go-openai library. Support GPT-4o, GPT-4o-mini, o1, o3-mini. Read API key from OPENAI_API_KEY env var. Map CompletionRequest to OpenAI chat completion API.
**Validate:** `cd /Users/rock/Rock/projects/council && export PATH="$HOME/go-sdk/go/bin:$PATH" && go build ./... && echo "ok"`

#### Task 2A.2: Gemini provider
**Files:** `internal/provider/gemini.go` (create)
**Do:** Implement Gemini provider via REST API (no official Go SDK needed, use net/http). Support gemini-2.5-pro, gemini-2.5-flash. Read API key from GEMINI_API_KEY env var. Endpoint: `https://generativelanguage.googleapis.com/v1beta/models/{model}:generateContent`.
**Validate:** `cd /Users/rock/Rock/projects/council && export PATH="$HOME/go-sdk/go/bin:$PATH" && go build ./... && echo "ok"`

#### Task 2A.3: Ollama provider
**Files:** `internal/provider/ollama.go` (create)
**Do:** Implement Ollama provider via REST API (localhost:11434). Support any model name. Endpoint: `POST http://localhost:11434/api/generate`. No API key needed. Add --ollama-host flag for custom host.
**Validate:** `cd /Users/rock/Rock/projects/council && export PATH="$HOME/go-sdk/go/bin:$PATH" && go build ./... && echo "ok"`

#### Task 2A.4: OpenRouter provider
**Files:** `internal/provider/openrouter.go` (create)
**Do:** Implement OpenRouter provider. Uses OpenAI-compatible API at `https://openrouter.ai/api/v1/chat/completions`. Read API key from OPENROUTER_API_KEY env var. This lets users access any model with a single API key.
**Validate:** `cd /Users/rock/Rock/projects/council && export PATH="$HOME/go-sdk/go/bin:$PATH" && go build ./... && echo "ok"`

### 2B: Additional Strategies

#### Task 2B.1: Vote strategy
**Files:** `internal/strategy/vote.go` (create)
**Do:** Implement vote strategy: each member reviews independently, then all responses are shown to the chair who identifies consensus points and majority positions. Simpler and cheaper than debate (no debate round). Good for factual questions.
**Validate:** `cd /Users/rock/Rock/projects/council && export PATH="$HOME/go-sdk/go/bin:$PATH" && go test ./internal/strategy/ -run TestVote -v`

#### Task 2B.2: Ranked strategy
**Files:** `internal/strategy/ranked.go` (create)
**Do:** Implement ranked strategy: each member reviews independently, then the chair does pairwise comparison of responses (which is better for each aspect?), producing a ranked synthesis. Most expensive but highest quality for complex questions.
**Validate:** `cd /Users/rock/Rock/projects/council && export PATH="$HOME/go-sdk/go/bin:$PATH" && go test ./internal/strategy/ -run TestRanked -v`

### 2C: Custom Councils

#### Task 2C.1: YAML council config
**Files:** `internal/config/config.go` (create), `councils/example.yaml` (create)
**Do:** Support loading custom council definitions from YAML files. Search for `.council.yaml` in current directory, then `~/.config/council/councils/`. Support `--config` flag for explicit path. Parse YAML into Council struct. Allow per-member provider/model overrides. Validate that referenced providers have API keys set.
**Validate:** `cd /Users/rock/Rock/projects/council && export PATH="$HOME/go-sdk/go/bin:$PATH" && go build -o council . && ./council list 2>&1 | grep -q "example" || echo "custom council not found (ok if no yaml)"`

### 2D: Probe Integration

#### Task 2D.1: Add --council flag to Probe
**Files:** Work in `/Users/rock/Rock/projects/code-review-agent/` — `cmd/review.go` (modify), add council dependency
**Do:** Add `--council` flag to Probe. When set, instead of single-model review, import council's code-review council and run deliberation. Map council's synthesized output back to Probe's Finding format. This replaces the single Claude call with a multi-perspective council review. Keep existing single-model review as default (council is opt-in).
**Validate:** `cd /Users/rock/Rock/projects/code-review-agent && export PATH="$HOME/go-sdk/go/bin:$PATH" && go build -o probe . && echo "func main() { fmt.Println(os.Getenv(\"SECRET\")) }" | ./probe --council --cli --stdin 2>&1 | head -30`

### 2E: Ship

#### Task 2E.1: README, goreleaser, deploy
**Files:** `README.md` (create), `.goreleaser.yml` (create), `.github/workflows/release.yml` (create)
**Do:** README with: problem statement ("one model has blind spots"), live demo (terminal recording placeholder), install instructions, usage examples showing all three councils, explanation of deliberation strategies, custom council YAML example, cost comparison (council vs single model), "what I learned" section. Goreleaser config for cross-platform binaries. Release workflow on tag push.
**Validate:** `cd /Users/rock/Rock/projects/council && export PATH="$HOME/go-sdk/go/bin:$PATH" && go build -o council . && ./council --help && cat README.md | head -5`

#### Task 2E.2: Create repo, push, tag
**Files:** n/a (git operations)
**Do:** Create GitHub repo `jtsilverman/council`. Push all code. Tag v0.1.0. Verify CI runs. Verify `go install github.com/jtsilverman/council@latest` works.
**Validate:** `gh repo view jtsilverman/council --json name,description`

## Phase 3: Stretch

### 3A: Advanced Features

#### Task 3A.1: ELO model ranking
**Files:** `internal/council/elo.go` (create)
**Do:** Track which members' findings survive the debate most often. Store ELO ratings in `~/.config/council/elo.json`. Update after each deliberation. Show rankings with `council stats`. Over time, this reveals which personas (and models) are most effective.
**Validate:** ELO file updates after a council run.

#### Task 3A.2: Deliberation replay
**Files:** `internal/output/replay.go` (create), `cmd/replay.go` (create)
**Do:** Save deliberation traces to `~/.config/council/traces/`. `council replay <trace-id>` re-renders the trace with full formatting. `council history` lists recent traces.
**Validate:** `./council history` shows recent runs.

#### Task 3A.3: MCP server mode
**Files:** `cmd/serve.go` (create)
**Do:** `council serve` runs as an MCP server, exposing `council_run` as a tool. Other AI agents can invoke council deliberations. Uses stdio transport.
**Validate:** Build succeeds and `./council serve --help` shows options.

## The One Hard Thing

**The debate prompt engineering.** The quality of the entire system depends on how well members debate each other's findings. Too agreeable = rubber stamp (no value over single model). Too adversarial = everything gets challenged and nothing survives.

Why it's hard:
- The debate prompt must encourage genuine critical thinking without degenerating into blanket disagreement
- Members need to maintain their persona perspective during debate (security auditor shouldn't challenge performance findings, they should challenge from a security angle)
- The chair synthesis prompt must correctly identify consensus vs contested findings and make good judgment calls on contested ones
- Different query types need different debate dynamics (factual questions need less debate, architectural questions need more)

Proposed approach:
- Debate prompt includes specific instructions: "Challenge findings ONLY if you have a specific technical reason. State your reason. Don't challenge findings outside your expertise unless they're clearly wrong."
- Chair prompt includes: "Findings supported by 2+ members are high confidence. Findings challenged with valid reasoning should be downgraded or dropped. Your job is to be the tiebreaker, not to add new findings."
- Calibrate by running the code-review council on known-buggy code (OWASP examples) and known-clean code. Measure false positive rate and true positive rate.

Fallback:
- If debate quality is poor, simplify to vote strategy (no debate round, just parallel review + chair synthesis). This still adds value over single-model review, just less.

## Risks

| Risk | Severity | Mitigation |
|------|----------|------------|
| Debate degenerates into agreement (no value added) | High | Persona prompts explicitly encourage dissent from their perspective. Measure agreement rate and tune. |
| Cost: 7-11 API calls in API mode | Medium | CLI mode is default ($0, uses subscription). API mode opt-in. Vote strategy is 4-6 calls. |
| Latency: multiple serial LLM calls | Medium | Parallel `claude --print` processes in CLI mode. Parallel goroutines in API mode. Only synthesis is serial. |
| API key management across providers | Low | Env vars per provider, OpenRouter as single-key fallback. Clear error messages for missing keys. |
| Karpathy's repo gets serious maintenance and eats the space | Low | Different language (Go vs Python), different form factor (CLI vs web app), different angle (designed personas vs model diversity). Complementary, not competing. |
| Probe integration adds complexity to Probe's codebase | Low | Council is an optional dependency, --council flag is opt-in. Probe works exactly as before without it. |

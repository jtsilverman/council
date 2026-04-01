# Council

Multi-perspective LLM deliberation CLI. Run any query through a council of expert personas that review independently, debate each other's findings, then a chair synthesizes the final answer.

**One model has blind spots. A council doesn't.**

## How It Works

```
council "Should we use Kafka or SQS for this event bus?"
```

```
Phase 1: Independent Review (parallel)
  ├── Analytical Thinker    → breaks down trade-offs
  ├── Creative Problem Solver → suggests unconventional alternatives
  └── Practical Engineer     → flags real-world gotchas

Phase 2: Debate (parallel)
  Each member challenges and supports each other's points

Phase 3: Chair Synthesis
  Moderator produces a decisive answer from consensus + contested points
```

Default mode uses `claude --print` ($0, subscription). No API key needed.

## Install

```bash
go install github.com/jtsilverman/council@latest
```

Or download a binary from [Releases](https://github.com/jtsilverman/council/releases).

## Quick Start

```bash
# Ask anything
council "What are the trade-offs of microservices vs monolith?"

# Code review
cat main.go | council --council code-review

# Light review (2 members, fast)
cat main.go | council --council code-review --light

# Deep review (4 members + debate)
cat main.go | council --council code-review --deep

# Review with all 10 experts
cat main.go | council --council code-review --all

# Pick specific experts
cat main.go | council --council code-review --with "security,concurrency,data"

# Review writing
cat draft.md | council --council writing
```

## Multi-Model Experiments

Council auto-detects providers from model names. Subscriptions first (free), APIs as fallback.

```bash
# Mix Claude + GPT + Gemini (each uses cheapest available method)
council --models "claude-opus-4-6,gpt-5.4,gemini-2.5-pro" "Your question"

# See what's available on your system
council providers
```

| Model prefix | Detection order |
|-------------|----------------|
| `claude-*` | `claude --print` (free) > Anthropic API |
| `gpt-*`, `o3`, `o4-*` | `codex exec` (free) > OpenAI API |
| `gemini-*` | `gemini` CLI (free) > Gemini API |
| `ollama:*` | Local HTTP (free) |
| `kimi-*` | Kimi API |
| `MiniMax-*` | MiniMax API |
| anything else | OpenRouter API (fallback) |

## Code Review Council (10 Members)

```bash
council members   # see all available members
```

| Member | Focus | Set |
|--------|-------|-----|
| `security` | Injection, auth bypass, data exposure, crypto | core, light |
| `bugs` | Logic errors, edge cases, nil dereferences, races | core, light |
| `performance` | Bottlenecks, allocations, N+1, caching | core |
| `maintainability` | Readability, abstraction, naming, coupling | core |
| `concurrency` | Race conditions, deadlocks, goroutine leaks | extended |
| `api` | Endpoints, contracts, versioning, compatibility | extended |
| `data` | SQL, migrations, transactions, cascades | extended |
| `errors` | Swallowed errors, retries, panic paths | extended |
| `deps` | Unused imports, deprecated packages, licenses | extended |
| `tests` | Coverage gaps, brittle tests, missing edge cases | extended |

Selection: `--light` (2), default (4), `--all` (10), `--with "name,name"` (pick).

## Deliberation Strategies

| Strategy | Phases | Speed | Best for |
|----------|--------|-------|----------|
| `vote` (default) | Review + Synthesis | Fast (~30s) | Most queries, code review |
| `debate` (`--deep`) | Review + Debate + Synthesis | Slower (~90s) | Complex questions, contested trade-offs |

## Custom Councils

```yaml
# .council.yaml
name: architecture-review
description: Review system architecture decisions
strategy: debate
members:
  - name: Distributed Systems Expert
    persona: "You specialize in distributed systems..."
  - name: Cost Optimizer
    persona: "You think about AWS bills..."
chair:
  name: VP of Engineering
  persona: "You weigh trade-offs..."
```

## Scan a Directory

```bash
# Light review of every source file
council scan .

# Deep review
council scan --deep ./src/
```

## The Hard Part

Debate prompt engineering. Too agreeable and you get a rubber stamp. Too adversarial and everything gets challenged into oblivion.

The solution: members challenge findings only within their expertise, with specific technical reasons. The chair resolves conflicts by keeping consensus findings and dropping successfully challenged ones.

## Inspired By

[Karpathy's llm-council](https://github.com/karpathy/llm-council) proved the concept (16K stars). Council is the production CLI: Go binary, pipe-friendly, multi-provider auto-detection, configurable strategies, 10 expert personas, custom councils.

## Tech Stack

- **Go 1.24** for fast single-binary CLI
- **cobra** for CLI framework
- **goroutines** for parallel member execution
- 8 provider backends (3 CLI subscription, 5 API)

## License

MIT

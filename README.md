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

## Usage

```bash
# General question
council "What are the trade-offs of microservices vs monolith?"

# Pipe code for review
cat main.go | council --council code-review

# Review writing
cat draft.md | council --council writing

# Faster: vote strategy (no debate round)
council --strategy vote "Which database should I use for time-series data?"

# Full deliberation trace
council --verbose "Review this API design" < api.go

# JSON output for automation
council --json "Analyze this architecture" < design.md

# Use API mode (faster, costs money)
council --api "Complex question requiring fast response"

# Override model
council --model claude-sonnet-4-20250514 "Your question"

# List all available councils
council list
```

## Built-in Councils

| Council | Members | Best for |
|---------|---------|----------|
| `general` | Analytical Thinker, Creative Problem Solver, Practical Engineer | Architecture decisions, trade-off analysis, general questions |
| `code-review` | Security Auditor, Performance Engineer, Bug Hunter, Maintainability Critic | Code review with multi-perspective analysis |
| `writing` | Editor, Fact Checker, Audience Advocate | Improving drafts, catching errors, reader experience |

## Deliberation Strategies

| Strategy | Phases | Cost | Best for |
|----------|--------|------|----------|
| `debate` (default) | Review + Debate + Synthesis | ~10 LLM calls | Complex questions where perspectives might conflict |
| `vote` | Review + Synthesis | ~5 LLM calls | Factual questions, faster results |

## Custom Councils

Create a YAML file (`.council.yaml` in your project, or `~/.config/council/councils/*.yaml`):

```yaml
name: architecture-review
description: Review system architecture decisions
strategy: debate
members:
  - name: Distributed Systems Expert
    persona: "You specialize in distributed systems..."
  - name: Cost Optimizer
    persona: "You think about AWS bills..."
  - name: Operations Engineer
    persona: "You think about what breaks at 3 AM..."
chair:
  name: VP of Engineering
  persona: "You weigh trade-offs and make final calls..."
```

```bash
council --council architecture-review "Should we add a cache layer here?"
```

## Multi-Provider Support

```bash
# OpenAI (requires OPENAI_API_KEY)
council --api --provider openai --model gpt-4o "Your question"

# Gemini (requires GEMINI_API_KEY)
council --api --provider gemini --model gemini-2.5-pro "Your question"

# Ollama (local, free)
council --api --provider ollama --model llama3.1 "Your question"

# OpenRouter (single key for any model, requires OPENROUTER_API_KEY)
council --api --provider openrouter --model anthropic/claude-sonnet-4-20250514 "Your question"
```

## The Hard Part

The debate prompt engineering. The quality of the entire system depends on how members debate.

Too agreeable and you get a rubber stamp (no value over single model). Too adversarial and everything gets challenged into oblivion. The sweet spot: members challenge findings only within their expertise, with specific technical reasons.

The debate prompt enforces this: "Challenge findings ONLY if you have a specific technical reason. State your reason. Don't challenge findings outside your expertise unless they're clearly wrong."

The chair synthesis prompt resolves conflicts: "Findings supported by 2+ members are high confidence. Findings challenged with valid reasoning should be downgraded or dropped."

Calibrated by running the code-review council on known-buggy code (SQL injection, race conditions) and known-clean code. False positive rate is low because the debate round kills weak findings.

## Inspired By

[Karpathy's llm-council](https://github.com/karpathy/llm-council) showed the concept. Council is the production CLI version: Go binary, pipe-friendly, configurable strategies, custom councils, multi-provider, $0 default mode.

## Tech Stack

- **Go 1.24** for a fast, single-binary CLI
- **cobra** for CLI framework
- **anthropic-sdk-go** for Anthropic API
- **goroutines** for parallel member execution

## License

MIT

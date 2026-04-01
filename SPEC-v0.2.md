# Council v0.2.0

## Overview

Refactor council from a working prototype into a polished product. Three major changes: (1) unified `council review` command that handles files, directories, and diffs automatically, (2) auto-detected provider routing so users just name models and council figures out whether to use CLI subscriptions or APIs, (3) extended code review roster with 10 expert members and easy opt-in/opt-out.

## Scope

### Building

- **Unified `council review` command** — replaces `scan`. Accepts files, directories, or `--diff` for staged changes. Auto-detects input type.
- **`--models` flag** — comma-separated model names. Council auto-detects provider per model (CLI subscription first, API fallback, OpenRouter catch-all).
- **`council providers` command** — shows detected CLI tools, API keys, and what's available.
- **`council members` command** — shows all available code review members with descriptions.
- **`--deep` flag** — full debate strategy. Default is vote (light/fast).
- **`--light` flag** — 2 members only (Security Auditor + Bug Hunter).
- **`--all` flag** — all 10 code review members.
- **`--with` flag** — pick specific members by name (e.g., `--with "security,concurrency,data"`).
- **Extended code review roster** — 10 members total. Core 4 (default), Light 2, plus 6 additional specialists.
- **Structured code review output** — findings format with file:line, severity, description, suggested fix. Not freeform essays.
- **Auto-detect provider from model name** — `claude-*` → CLI, `gpt-*`/`o1`/`o3` → codex exec or OpenAI API, `gemini-*` → gemini CLI or Gemini API, `ollama:*` → local HTTP.
- **Bug fixes from self-scan** — Gemini API key in header (not URL), dynamic pricing, provider call timeouts, file exclusions for scan, ANSI sanitization.
- **Updated README**

### Not building

- Per-member model assignment via CLI flags (e.g., `--security claude --bugs gpt-4o`). YAML config covers this for power users.
- ELO ranking, deliberation replay, MCP server (Phase 3, future).
- Probe integration (separate repo, future).
- New general or writing council members (code review focus for v0.2.0).

### Ship target

GitHub release v0.2.0 via goreleaser. `go install` updated.

## Stack

Same as v0.1.0. Go 1.24, cobra, anthropic-sdk-go, yaml.v3. No new dependencies.

## Architecture

### Changes to existing files

```
cmd/
├── root.go          (modify — replace --api/--provider/--members with --models/--deep/--light/--all/--with)
├── run.go           (modify — update provider creation to use auto-detect)
├── scan.go          (delete — replaced by review.go)
├── review.go        (create — unified review: files, dirs, diffs)
├── providers.go     (create — council providers command)
├── members.go       (create — council members command)
internal/
├── provider/
│   ├── detect.go    (create — auto-detect provider from model name)
│   ├── gemini.go    (modify — API key in header, not URL)
│   ├── anthropic.go (modify — dynamic pricing by model)
│   ├── codex.go     (create — codex exec provider for GPT/o1/o3 models)
│   └── provider.go  (modify — add context timeout wrapper)
├── persona/
│   ├── codereview.go (modify — expand to 10 members, add roster registry)
│   └── roster.go    (create — member lookup by short name, --with parsing)
├── output/
│   ├── terminal.go  (modify — ANSI sanitization, structured review format)
│   └── findings.go  (create — structured Finding type for code review output)
├── strategy/
│   └── debate.go    (modify — structured output prompt for code review council)
```

### Auto-detect Provider Logic

```go
func DetectProvider(modelName string) (Provider, error) {
    // 1. Claude models → claude --print (free)
    if isClaudeModel(modelName) {
        if cliAvailable("claude") { return NewCLIProvider(modelName), nil }
        if hasEnv("ANTHROPIC_API_KEY") { return NewAnthropicProvider() }
    }
    // 2. GPT/o1/o3 models → codex exec (free) or OpenAI API
    if isOpenAIModel(modelName) {
        if cliAvailable("codex") { return NewCodexProvider(modelName), nil }
        if hasEnv("OPENAI_API_KEY") { return NewOpenAIProvider() }
    }
    // 3. Gemini models → gemini CLI (free) or Gemini API
    if isGeminiModel(modelName) {
        if cliAvailable("gemini") { return NewGeminiCLIProvider(modelName), nil }
        if hasEnv("GEMINI_API_KEY") { return NewGeminiProvider() }
    }
    // 4. Ollama models → local HTTP
    if isOllamaModel(modelName) { return NewOllamaProvider(""), nil }
    // 5. OpenRouter fallback
    if hasEnv("OPENROUTER_API_KEY") { return NewOpenRouterProvider() }
    // 6. Error
    return nil, fmt.Errorf("no provider for model %q. Run 'council providers' to see what's available", modelName)
}
```

### Extended Code Review Roster

| Short Name | Full Name | Focus | Default Set |
|------------|-----------|-------|-------------|
| `security` | Security Auditor | Injection, auth, data exposure, crypto | Core 4, Light 2 |
| `bugs` | Bug Hunter | Logic errors, edge cases, nil derefs, races | Core 4, Light 2 |
| `performance` | Performance Engineer | Bottlenecks, allocations, N+1, caching | Core 4 |
| `maintainability` | Maintainability Critic | Readability, abstraction, naming, coupling | Core 4 |
| `concurrency` | Concurrency Reviewer | Races, deadlocks, goroutine leaks, atomics | Extended |
| `api` | API Designer | Endpoints, contracts, versioning, compatibility | Extended |
| `data` | Data Integrity Checker | SQL, migrations, transactions, cascades | Extended |
| `errors` | Error Handling Auditor | Swallowed errors, retries, panic paths | Extended |
| `deps` | Dependency Auditor | Unused imports, deprecated packages, licenses | Extended |
| `tests` | Test Strategist | Coverage gaps, brittle tests, missing edge cases | Extended |

### Structured Finding Format

```go
type Finding struct {
    File        string `json:"file"`
    Line        int    `json:"line,omitempty"`
    Severity    string `json:"severity"`    // "critical", "high", "medium", "low"
    Category    string `json:"category"`    // "security", "bug", "performance", etc.
    Title       string `json:"title"`
    Description string `json:"description"`
    Suggestion  string `json:"suggestion,omitempty"`
}
```

The chair's synthesis prompt will instruct it to output findings in this structured format (JSON block in the response), which council parses into Finding structs for clean terminal rendering.

### Review Command Input Detection

```
council review main.go           → single file review
council review main.go util.go   → multi-file review
council review .                 → directory scan (walk + review each file)
council review ./src/            → directory scan
council review --diff            → git diff --cached | review
council review --diff main       → git diff main...HEAD | review
```

## Task List

## Phase 1: Provider Auto-Detection

### Task 1.1: Provider detection engine
**Files:** `internal/provider/detect.go` (create), `internal/provider/codex.go` (create)
**Do:** Implement `DetectProvider(modelName string) (Provider, error)` with the priority logic from the architecture section. Add helper functions: `isClaudeModel()`, `isOpenAIModel()`, `isGeminiModel()`, `isOllamaModel()`, `cliAvailable()`, `hasEnv()`. Implement `CodexProvider` that shells out to `codex exec` for GPT/o1/o3 models (similar pattern to CLIProvider but using codex). Add `GeminiCLIProvider` that shells out to `echo "prompt" | gemini` for Gemini models.
**Validate:** `cd /Users/rock/Rock/projects/council && export PATH="$HOME/go-sdk/go/bin:$PATH" && go build ./... && echo "ok"`

### Task 1.2: Providers command
**Files:** `cmd/providers.go` (create)
**Do:** Add `council providers` subcommand. Check each provider: CLI tools (claude, codex, gemini via `exec.LookPath`), API keys (ANTHROPIC_API_KEY, OPENAI_API_KEY, GEMINI_API_KEY, OPENROUTER_API_KEY via `os.Getenv`), Ollama (HTTP ping to localhost:11434). Display a table with status (available/not found) and what models each enables.
**Validate:** `cd /Users/rock/Rock/projects/council && export PATH="$HOME/go-sdk/go/bin:$PATH" && go build -o council . && ./council providers`

### Task 1.3: Wire --models flag into run command
**Files:** `cmd/root.go` (modify), `cmd/run.go` (modify)
**Do:** Add `--models` flag (comma-separated string). When provided, create a provider per model using `DetectProvider`. Update the council run logic: if `--models` is set, assign each model's provider to the corresponding council member (round-robin if more members than models). Remove the `--api` and `--provider` flags (replaced by auto-detection). Keep `--model` as a single-model override for all members.
**Validate:** `cd /Users/rock/Rock/projects/council && export PATH="$HOME/go-sdk/go/bin:$PATH" && go build -o council . && ./council --models "claude-sonnet-4-20250514" "Say hello" 2>&1 | head -5`

## Phase 2: Extended Roster & Member Selection

### Task 2.1: Extended code review roster
**Files:** `internal/persona/codereview.go` (modify), `internal/persona/roster.go` (create)
**Do:** Add 6 new members to the code review council: Concurrency Reviewer, API Designer, Data Integrity Checker, Error Handling Auditor, Dependency Auditor, Test Strategist. Each with a carefully crafted persona prompt. Create `roster.go` with: `AllMembers()` returning all 10, `CoreMembers()` returning the default 4, `LightMembers()` returning Security + Bug Hunter, `GetMembersByNames(names []string)` for --with flag lookup. Short names: security, bugs, performance, maintainability, concurrency, api, data, errors, deps, tests.
**Validate:** `cd /Users/rock/Rock/projects/council && export PATH="$HOME/go-sdk/go/bin:$PATH" && go build ./... && echo "ok"`

### Task 2.2: Member selection flags and members command
**Files:** `cmd/root.go` (modify), `cmd/run.go` (modify), `cmd/members.go` (create)
**Do:** Add flags: `--light` (2 members), `--deep` (debate + core 4), `--all` (all 10), `--with "name,name"` (pick specific). These only apply to the code-review council. Add `council members` subcommand that lists all 10 members with short name, full name, and one-line description. Flag priority: --with > --all > --light > --members N > default (core 4).
**Validate:** `cd /Users/rock/Rock/projects/council && export PATH="$HOME/go-sdk/go/bin:$PATH" && go build -o council . && ./council members`

## Phase 3: Unified Review Command

### Task 3.1: Review command with input detection
**Files:** `cmd/review.go` (create), `cmd/scan.go` (delete)
**Do:** Create `council review` subcommand. Input detection: if args are files, review each file. If arg is a directory, walk and review all source files (reuse scan logic). If `--diff` flag, run `git diff --cached` (or `git diff <branch>...HEAD` if branch specified) and review the diff. Remove the old `scan` command. Review uses code-review council by default. For directory mode, process files with concurrency limit of 2. Skip files >50KB, skip .env/.pem/.key files.
**Validate:** `cd /Users/rock/Rock/projects/council && export PATH="$HOME/go-sdk/go/bin:$PATH" && go build -o council . && ./council review --help`

### Task 3.2: Structured code review output
**Files:** `internal/output/findings.go` (create), `internal/output/terminal.go` (modify), `internal/strategy/debate.go` (modify), `internal/strategy/vote.go` (modify)
**Do:** Define `Finding` struct (file, line, severity, category, title, description, suggestion). Update the chair synthesis prompt for code-review council to instruct it to output findings as a JSON array in a code block, followed by a summary. Parse the JSON from the chair's response into `[]Finding`. Render findings in terminal with colored severity badges, file:line references, and suggested fixes. Fall back to raw text if JSON parsing fails (LLM output isn't always clean). For non-code-review councils, keep the current freeform output.
**Validate:** `cd /Users/rock/Rock/projects/council && export PATH="$HOME/go-sdk/go/bin:$PATH" && go build -o council . && echo 'func handler(w http.ResponseWriter, r *http.Request) { db.Query("SELECT * FROM users WHERE id=" + r.URL.Query().Get("id")) }' | ./council review --light 2>&1 | head -30`

## Phase 4: Bug Fixes

### Task 4.1: Gemini API key in header
**Files:** `internal/provider/gemini.go` (modify)
**Do:** Move the API key from URL query parameter to `x-goog-api-key` HTTP header. This prevents key leakage in logs and proxy servers.
**Validate:** `cd /Users/rock/Rock/projects/council && export PATH="$HOME/go-sdk/go/bin:$PATH" && go build ./... && grep -c "key=" internal/provider/gemini.go` returns 0

### Task 4.2: Dynamic pricing, timeouts, file exclusions, ANSI sanitization
**Files:** `internal/provider/anthropic.go` (modify), `internal/provider/provider.go` (modify), `cmd/review.go` (modify), `internal/output/terminal.go` (modify)
**Do:** (1) Anthropic pricing: look up cost per model instead of hardcoding Sonnet rates. (2) Add `context.WithTimeout` wrapper (120s default) in provider.go that all providers use. (3) In the review command's directory mode, skip files matching `.env`, `*.pem`, `*.key`, `*.cert`, `credentials*`, `*secret*`. (4) Strip ANSI escape sequences from LLM response content before rendering (regex: `\x1b\[[0-9;]*[a-zA-Z]`).
**Validate:** `cd /Users/rock/Rock/projects/council && export PATH="$HOME/go-sdk/go/bin:$PATH" && go build ./... && echo "ok"`

## Phase 5: Ship

### Task 5.1: Update README
**Files:** `README.md` (modify)
**Do:** Rewrite README for v0.2.0: updated usage examples showing `council review`, `--models`, `--light/--deep/--all/--with`, `council providers`, `council members`. Show the extended roster table. Document auto-provider detection. Remove old `scan` command docs. Keep "The Hard Part" and "Inspired By" sections.
**Validate:** `cat README.md | head -5` shows correct title

### Task 5.2: Tag and release
**Files:** n/a (git operations)
**Do:** Commit all changes. Tag v0.2.0. Push. Verify goreleaser builds. Update `go install` instructions. Install updated binary on Mac Mini.
**Validate:** `cd /Users/rock/Rock/projects/council && export PATH="$HOME/go-sdk/go/bin:$PATH" && go build -o council . && ./council --help && ./council providers && ./council members`

## The One Hard Thing

**Structured output from the chair synthesis.** The chair needs to produce findings as parseable JSON, not freeform text. LLMs don't reliably output valid JSON, especially after processing a long debate context. The prompt engineering must be tight enough that the JSON is parseable 90%+ of the time, with a graceful fallback to raw text when it isn't.

Approach: Instruct the chair to wrap findings in a ```json code block. Parse the first JSON array found in the response. If parsing fails, display the raw synthesis text (still useful, just not structured).

Fallback: Skip structured parsing entirely and just display the chair's freeform synthesis (current behavior). Still valuable, just less polished.

## Risks

| Risk | Severity | Mitigation |
|------|----------|------------|
| Codex exec may not work like claude --print | Medium | Test during implementation. If codex doesn't support piped prompts the same way, fall back to OpenAI API. |
| Structured JSON output from LLM is unreliable | Medium | Graceful fallback to freeform text. Parse best-effort. |
| 10 members + debate = very slow | Low | All 10 is opt-in (--all). Default is 4 (vote). Light is 2. User controls the tradeoff. |
| Removing --api flag is a breaking change | Low | v0.2.0 is a minor bump. The replacement (--models or auto-detect) is strictly better. |

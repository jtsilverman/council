package strategy

import (
	"os"
	"strings"
	"testing"

	"github.com/jtsilverman/council/internal/provider"
	"github.com/jtsilverman/council/internal/review"
)

func TestBuildReviewPrompt_Default(t *testing.T) {
	profile := provider.DefaultProfile()
	result := BuildReviewPrompt(profile, "Review this code:\nfunc main() {}")

	// Default profile should NOT add output contract
	if strings.Contains(result, "VERDICT:") && strings.Contains(result, "Respond in this exact format") {
		t.Error("default profile should not add output contract")
	}
	if !strings.Contains(result, "Review this code:") {
		t.Error("should contain query")
	}
}

func TestBuildReviewPrompt_Codex(t *testing.T) {
	profile := provider.CodexProfile()
	query := "diff --git a/main.go b/main.go\n--- a/main.go\n+++ b/main.go\n@@ -1,3 +1,4 @@\n package main\n+import \"fmt\"\n func main() {}"
	result := BuildReviewPrompt(profile, query)

	// Codex profile should add output contract
	if !strings.Contains(result, "Respond in this exact format") {
		t.Error("codex profile should add output contract")
	}
	if !strings.Contains(result, "VERDICT: PASS | CONCERN | FAIL") {
		t.Error("codex profile should include verdict format")
	}

	// Codex profile should rewrite for workspace
	if !strings.Contains(result, "Read the full files from the workspace") {
		t.Error("codex profile should rewrite query for workspace")
	}
	if !strings.Contains(result, "Key files:") {
		t.Error("codex profile should list key files")
	}
	if !strings.Contains(result, "main.go") {
		t.Error("should contain extracted path main.go")
	}
}

func TestBuildReviewPrompt_QueryTruncated(t *testing.T) {
	profile := provider.CodexRetryProfile() // 4000 char budget
	longQuery := strings.Repeat("review this code carefully ", 500)

	result := BuildReviewPrompt(profile, longQuery)

	if len(result) > profile.ReviewBudgetChars {
		t.Errorf("result too long: %d chars, budget %d", len(result), profile.ReviewBudgetChars)
	}
}

func TestBuildDebatePrompt(t *testing.T) {
	profile := provider.DefaultProfile()
	digests := []review.ReviewDigest{
		{
			Verdict:  "CONCERN",
			Findings: []review.FindingNote{{Severity: "high", Title: "Missing auth", File: "api.go", Line: "10"}},
			Summary:  "Auth issue found.",
		},
		{
			Verdict:  "PASS",
			Findings: []review.FindingNote{{Severity: "low", Title: "Typo", File: "main.go"}},
			Summary:  "Looks good.",
		},
	}

	result := BuildDebatePrompt(profile, "review the PR", digests)

	if !strings.Contains(result, "PHASE 1 FINDINGS") {
		t.Error("should contain phase 1 findings header")
	}
	if !strings.Contains(result, "Missing auth") {
		t.Error("should contain finding from digest")
	}
	if !strings.Contains(result, "CHALLENGE") {
		t.Error("should contain debate instructions")
	}
	if !strings.Contains(result, "Original query:") {
		t.Error("should contain original query")
	}
}

func TestBuildDebatePrompt_Codex(t *testing.T) {
	profile := provider.CodexProfile()
	digests := []review.ReviewDigest{
		{
			Verdict:  "FAIL",
			Findings: []review.FindingNote{{Severity: "critical", Title: "SQL injection"}},
			Summary:  "Critical issue.",
		},
	}

	result := BuildDebatePrompt(profile, "review", digests)

	// Should include output contract for codex
	if !strings.Contains(result, "Respond in this exact format") {
		t.Error("codex debate should include output contract")
	}
}

func TestBuildSynthesisPrompt(t *testing.T) {
	profile := provider.DefaultProfile()
	allDigests := []review.ReviewDigest{
		{
			Verdict:  "CONCERN",
			Findings: []review.FindingNote{{Severity: "high", Title: "Auth bypass"}},
			Summary:  "Security issue.",
		},
		{
			Verdict:  "PASS",
			Findings: []review.FindingNote{{Severity: "medium", Title: "Missing docs"}},
			Summary:  "Minor issues.",
		},
		{
			Verdict:  "CONCERN",
			Findings: []review.FindingNote{{Severity: "high", Title: "Auth bypass confirmed"}},
			Summary:  "Agree with reviewer 1.",
		},
	}

	result := BuildSynthesisPrompt(profile, "review the PR", allDigests)

	if !strings.Contains(result, "ALL FINDINGS") {
		t.Error("should contain all findings header")
	}
	if !strings.Contains(result, "Auth bypass") {
		t.Error("should contain finding text")
	}
	if !strings.Contains(result, "Synthesize a final response") {
		t.Error("should contain synthesis instructions")
	}
	if !strings.Contains(result, "consensus support") {
		t.Error("should contain consensus rule")
	}
}

func TestCompactDigests_UnderBudget(t *testing.T) {
	digests := []review.ReviewDigest{
		{
			Verdict:  "PASS",
			Findings: []review.FindingNote{{Severity: "low", Title: "Minor thing"}},
			Summary:  "All good.",
		},
	}

	result := CompactDigests(digests, 10_000)

	// Should contain all content unchanged
	if !strings.Contains(result, "VERDICT: PASS") {
		t.Error("should contain verdict")
	}
	if !strings.Contains(result, "Minor thing") {
		t.Error("should contain finding")
	}
	if !strings.Contains(result, "All good.") {
		t.Error("should contain summary")
	}
}

func TestCompactDigests_OverBudget(t *testing.T) {
	var findings []review.FindingNote
	// Create many findings to exceed budget
	findings = append(findings, review.FindingNote{Severity: "critical", Title: "SQL injection", Rationale: "User input not sanitized in database query"})
	findings = append(findings, review.FindingNote{Severity: "high", Title: "Auth bypass", Rationale: "Token validation skipped on admin endpoints"})
	for i := 0; i < 20; i++ {
		findings = append(findings, review.FindingNote{
			Severity:  "low",
			Title:     strings.Repeat("Low severity issue description ", 5),
			Rationale: strings.Repeat("Detailed rationale for this finding. ", 10),
		})
	}

	digests := []review.ReviewDigest{
		{
			Verdict:  "FAIL",
			Findings: findings,
			Summary:  strings.Repeat("Long summary text. ", 50),
		},
	}

	// Use a tight budget
	result := CompactDigests(digests, 500)

	if len(result) > 500 {
		t.Errorf("result length %d exceeds budget 500", len(result))
	}

	// Critical and high should be preserved (if they fit)
	if !strings.Contains(result, "critical") && !strings.Contains(result, "SQL injection") {
		t.Log("Note: critical finding was truncated by hard budget limit")
	}
}

func TestCompactDigests_KeepsCriticalHigh(t *testing.T) {
	findings := []review.FindingNote{
		{Severity: "critical", Title: "SQL injection"},
		{Severity: "high", Title: "Auth bypass"},
		{Severity: "medium", Title: "Missing docs"},
		{Severity: "low", Title: "Typo"},
		{Severity: "low", Title: "Style issue"},
		{Severity: "low", Title: "Naming convention"},
	}

	digests := []review.ReviewDigest{
		{Verdict: "FAIL", Findings: findings, Summary: "Multiple issues."},
	}

	// Budget that fits trimmed but not full
	full := review.FormatDigest(digests[0])
	trimmedDigest := review.TrimFindings(digests[0], 3)
	trimmedStr := review.FormatDigest(trimmedDigest)

	budget := (len(full) + len(trimmedStr)) / 2 // between full and trimmed
	result := CompactDigests(digests, budget)

	// Should keep critical and high
	if !strings.Contains(result, "SQL injection") {
		t.Error("should keep critical finding")
	}
	if !strings.Contains(result, "Auth bypass") {
		t.Error("should keep high finding")
	}
}

func TestRewriteQueryForWorkspace(t *testing.T) {
	query := `Here's the code to review:

diff --git a/internal/server.go b/internal/server.go
--- a/internal/server.go
+++ b/internal/server.go
@@ -10,6 +10,7 @@
 func main() {
+    fmt.Println("hello")
 }

diff --git a/cmd/root.go b/cmd/root.go
--- a/cmd/root.go
+++ b/cmd/root.go
@@ -1,3 +1,4 @@
 package cmd
+import "os"
`

	result := RewriteQueryForWorkspace(query)

	if !strings.Contains(result, "Read the full files from the workspace") {
		t.Error("should add workspace instruction")
	}
	if !strings.Contains(result, "Key files:") {
		t.Error("should list key files")
	}
	if !strings.Contains(result, "internal/server.go") {
		t.Error("should extract internal/server.go")
	}
	if !strings.Contains(result, "cmd/root.go") {
		t.Error("should extract cmd/root.go")
	}
	// Should still contain the diff
	if !strings.Contains(result, "diff --git") {
		t.Error("should preserve the diff content")
	}
}

func TestRewriteQueryForWorkspace_NoDiff(t *testing.T) {
	query := "Review the function handleRequest in server.go for potential issues."

	result := RewriteQueryForWorkspace(query)

	if result != query {
		t.Errorf("should return query unchanged when no diff, got: %s", result)
	}
}

// =============================================================================
// Regression tests: end-to-end budget verification with large fixtures
// =============================================================================

// Manual smoke test:
// go test ./... && printf '%s\n' "$(cat testdata/large_review.txt)" | \
//   go run . --council code-review --models claude-sonnet-4-6,gpt-5.4,gemini-2.5-pro --deep

func loadFixture(t *testing.T) string {
	t.Helper()
	data, err := os.ReadFile("../../testdata/large_review.txt")
	if err != nil {
		t.Fatalf("failed to load fixture: %v", err)
	}
	if len(data) < 20_000 {
		t.Fatalf("fixture too small: %d chars, expected >20K", len(data))
	}
	return string(data)
}

func TestBudgetRegression_CodexReviewUnderBudget(t *testing.T) {
	fixture := loadFixture(t)
	profile := provider.CodexProfile()

	// BuildReviewPrompt applies workspace rewrite for Codex (PreferWorkspaceRead=true),
	// which strips pasted file contents and keeps only the diff + file list.
	// Persona is now passed separately via SystemPrompt, not embedded in the prompt.
	result := BuildReviewPrompt(profile, fixture)

	if len(result) >= profile.ReviewBudgetChars {
		t.Errorf("Codex review prompt is %d chars, exceeds budget of %d chars",
			len(result), profile.ReviewBudgetChars)
	}
	t.Logf("Codex review prompt: %d / %d chars (%.0f%% of budget)",
		len(result), profile.ReviewBudgetChars,
		float64(len(result))/float64(profile.ReviewBudgetChars)*100)
}

func TestBudgetRegression_CodexDebateUnderBudget(t *testing.T) {
	fixture := loadFixture(t)
	profile := provider.CodexProfile()

	// In the real pipeline, the query is workspace-rewritten before reaching debate.
	// Apply the same transformation here to test realistic budget behavior.
	rewrittenQuery := RewriteQueryForWorkspace(fixture)

	// Simulate 3 panel members with multiple findings each
	digests := []review.ReviewDigest{
		{
			Verdict: "CONCERN",
			Findings: []review.FindingNote{
				{Severity: "high", File: "internal/server/handler.go", Line: "28", Title: "No input validation on request body", Rationale: "The handler reads the entire body without size checks beyond maxBody. JSON decode errors leak internal structure."},
				{Severity: "medium", File: "internal/middleware/chain.go", Line: "62", Title: "Rate limiter not thread-safe under high load", Rationale: "Token bucket refill races with concurrent requests. The mutex protects individual ops but refill+check is not atomic."},
				{Severity: "low", File: "cmd/root.go", Line: "15", Title: "Global vars for CLI flags", Rationale: "Using package-level vars for cobra flags makes testing harder. Consider a config struct."},
			},
			Summary: "Handler needs input validation hardening. Rate limiter has a subtle race.",
		},
		{
			Verdict: "FAIL",
			Findings: []review.FindingNote{
				{Severity: "critical", File: "internal/provider/router.go", Line: "45", Title: "Circuit breaker check is racy", Rationale: "Reading health under RLock then writing under Lock creates a TOCTOU race. Provider could be used after circuit opens."},
				{Severity: "high", File: "internal/provider/codex.go", Line: "35", Title: "Retry prompt truncation loses context", Rationale: "Cutting at 60% and finding last newline can split in the middle of a diff hunk, producing invalid diff that confuses the model."},
				{Severity: "medium", File: "internal/strategy/strategy.go", Line: "72", Title: "Error slice not checked before proceeding", Rationale: "runParallelReviews returns errors but Execute never checks them. Could synthesize from partial/empty results."},
			},
			Summary: "Critical race in provider router. Codex retry truncation is too aggressive.",
		},
		{
			Verdict: "CONCERN",
			Findings: []review.FindingNote{
				{Severity: "high", File: "internal/server/router.go", Line: "30", Title: "No authentication middleware on review endpoint", Rationale: "The /api/v1/review endpoint has rate limiting but no auth. Anyone can submit reviews consuming API credits."},
				{Severity: "medium", File: "internal/strategy/vote.go", Line: "40", Title: "decideFinalVerdict logic is fragile", Rationale: "The verdict cascade (FAIL > CONCERN > PASS) doesn't handle edge cases like all-unknown or mixed unknown+pass."},
				{Severity: "low", File: "cmd/review.go", Line: "25", Title: "Router initialized with nil providers", Rationale: "NewProviderRouter(nil) creates empty router, then Route will always fail. Dead code path."},
			},
			Summary: "Missing auth on API endpoint is the biggest gap. Vote logic needs edge case handling.",
		},
	}

	result := BuildDebatePrompt(profile, rewrittenQuery, digests)

	if len(result) >= profile.DebateBudgetChars {
		t.Errorf("Codex debate prompt is %d chars, exceeds budget of %d chars",
			len(result), profile.DebateBudgetChars)
	}
	t.Logf("Codex debate prompt: %d / %d chars (%.0f%% of budget)",
		len(result), profile.DebateBudgetChars,
		float64(len(result))/float64(profile.DebateBudgetChars)*100)
}

func TestBudgetRegression_CodexSynthesisUnderBudget(t *testing.T) {
	fixture := loadFixture(t)
	profile := provider.CodexProfile()

	// In the real pipeline, the query is workspace-rewritten before reaching synthesis.
	rewrittenQuery := RewriteQueryForWorkspace(fixture)

	// 3 review digests + 3 debate digests = 6 total
	allDigests := []review.ReviewDigest{
		// Phase 1: review
		{
			Verdict: "CONCERN",
			Findings: []review.FindingNote{
				{Severity: "high", File: "internal/server/handler.go", Line: "28", Title: "No input validation"},
				{Severity: "medium", File: "internal/middleware/chain.go", Line: "62", Title: "Rate limiter race"},
			},
			Summary: "Handler needs validation hardening.",
		},
		{
			Verdict: "FAIL",
			Findings: []review.FindingNote{
				{Severity: "critical", File: "internal/provider/router.go", Line: "45", Title: "Circuit breaker TOCTOU"},
				{Severity: "high", File: "internal/provider/codex.go", Line: "35", Title: "Retry truncation loses context"},
			},
			Summary: "Critical race in provider router.",
		},
		{
			Verdict: "CONCERN",
			Findings: []review.FindingNote{
				{Severity: "high", File: "internal/server/router.go", Line: "30", Title: "No auth on review endpoint"},
				{Severity: "medium", File: "internal/strategy/vote.go", Line: "40", Title: "Fragile verdict logic"},
			},
			Summary: "Missing auth is biggest gap.",
		},
		// Phase 2: debate
		{
			Verdict: "FAIL",
			Findings: []review.FindingNote{
				{Severity: "critical", File: "internal/provider/router.go", Line: "45", Title: "Confirmed: circuit breaker race is real"},
				{Severity: "high", File: "internal/server/handler.go", Line: "28", Title: "Agree: input validation missing"},
			},
			Summary: "Circuit breaker race confirmed by all members.",
		},
		{
			Verdict: "CONCERN",
			Findings: []review.FindingNote{
				{Severity: "medium", File: "internal/middleware/chain.go", Line: "62", Title: "Rate limiter race is theoretical, not practical"},
				{Severity: "high", File: "internal/server/router.go", Line: "30", Title: "Auth gap confirmed"},
			},
			Summary: "Rate limiter issue downgraded. Auth gap confirmed.",
		},
		{
			Verdict: "FAIL",
			Findings: []review.FindingNote{
				{Severity: "high", File: "internal/provider/codex.go", Line: "35", Title: "Challenged: truncation is acceptable tradeoff"},
				{Severity: "critical", File: "internal/provider/router.go", Line: "45", Title: "Strong support: circuit breaker must be fixed"},
			},
			Summary: "Router race is consensus critical. Truncation is debatable.",
		},
	}

	result := BuildSynthesisPrompt(profile, rewrittenQuery, allDigests)

	if len(result) >= profile.SynthesisBudgetChars {
		t.Errorf("Codex synthesis prompt is %d chars, exceeds budget of %d chars",
			len(result), profile.SynthesisBudgetChars)
	}
	t.Logf("Codex synthesis prompt: %d / %d chars (%.0f%% of budget)",
		len(result), profile.SynthesisBudgetChars,
		float64(len(result))/float64(profile.SynthesisBudgetChars)*100)
}

func TestParseDigest_MultiFormat(t *testing.T) {
	// Opus-style: markdown with ### Verdict, numbered [severity: X] findings
	opusRaw := `### Verdict: PASS

1. [severity: medium] Missing error wrapping in handler
   The error from json.Decode is returned raw without context.

2. [severity: low] Unused import in router.go
   The "log" import is declared but only used in one branch.

### Summary: Generally clean code with minor error handling gaps.`

	// Codex-style: clean structured VERDICT/FINDING/SUMMARY
	codexRaw := `VERDICT: CONCERN
FINDING: high | internal/server/handler.go:28 | No request size validation | Body is read with LimitReader but limit is 1MB which may be too generous for review payloads
FINDING: medium | internal/middleware/chain.go:62 | Token bucket not fully atomic | Refill and decrement should be a single critical section
SUMMARY: Handler accepts oversized payloads and rate limiter has minor atomicity gap.`

	// Gemini-style: freeform text with no structured markers
	geminiRaw := `The code looks reasonable overall. I noticed a few things:

The handler in server/handler.go doesn't validate the content type header before
attempting JSON decode. This could lead to confusing error messages for clients
sending form data or plain text.

The middleware chain applies Recovery before Logger, which means panics won't
be logged with request context. Consider swapping the order.

The provider router's health tracking seems incomplete - circuitOpen is set
based on failures but there's no mechanism to close it again after recovery.

Overall this is a solid PR with some minor improvements needed.`

	tests := []struct {
		name string
		raw  string
	}{
		{"opus", opusRaw},
		{"codex", codexRaw},
		{"gemini", geminiRaw},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			digest := review.ParseDigest(tt.raw)

			if digest.Verdict == "" {
				t.Error("verdict should not be empty")
			}
			if len(digest.Findings) == 0 {
				t.Error("should have at least one finding")
			}

			t.Logf("%s: verdict=%s, findings=%d, summary_len=%d",
				tt.name, digest.Verdict, len(digest.Findings), len(digest.Summary))
		})
	}

	// Verify specific parsing for each format
	t.Run("opus_specifics", func(t *testing.T) {
		digest := review.ParseDigest(opusRaw)
		if digest.Verdict != "PASS" {
			t.Errorf("opus verdict should be PASS, got %q", digest.Verdict)
		}
		if len(digest.Findings) < 2 {
			t.Errorf("opus should have at least 2 findings, got %d", len(digest.Findings))
		}
	})

	t.Run("codex_specifics", func(t *testing.T) {
		digest := review.ParseDigest(codexRaw)
		if digest.Verdict != "CONCERN" {
			t.Errorf("codex verdict should be CONCERN, got %q", digest.Verdict)
		}
		if len(digest.Findings) != 2 {
			t.Errorf("codex should have 2 findings, got %d", len(digest.Findings))
		}
		if digest.Summary == "" {
			t.Error("codex should have a summary")
		}
	})

	t.Run("gemini_specifics", func(t *testing.T) {
		digest := review.ParseDigest(geminiRaw)
		// Gemini freeform should hit fallback: verdict=unknown, one finding with raw text
		if digest.Verdict != "unknown" {
			t.Errorf("gemini freeform should produce unknown verdict, got %q", digest.Verdict)
		}
		if len(digest.Findings) == 0 {
			t.Error("gemini freeform should produce at least one fallback finding")
		}
	})
}

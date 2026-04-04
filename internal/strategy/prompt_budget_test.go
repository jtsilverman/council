package strategy

import (
	"strings"
	"testing"

	"github.com/jtsilverman/council/internal/provider"
	"github.com/jtsilverman/council/internal/review"
)

func TestBuildReviewPrompt_Default(t *testing.T) {
	profile := provider.DefaultProfile()
	result := BuildReviewPrompt(profile, "You are a security expert.", "Review this code:\nfunc main() {}")

	// Default profile should NOT add output contract
	if strings.Contains(result, "VERDICT:") && strings.Contains(result, "Respond in this exact format") {
		t.Error("default profile should not add output contract")
	}
	if !strings.Contains(result, "You are a security expert.") {
		t.Error("should contain persona")
	}
	if !strings.Contains(result, "Review this code:") {
		t.Error("should contain query")
	}
}

func TestBuildReviewPrompt_Codex(t *testing.T) {
	profile := provider.CodexProfile()
	query := "diff --git a/main.go b/main.go\n--- a/main.go\n+++ b/main.go\n@@ -1,3 +1,4 @@\n package main\n+import \"fmt\"\n func main() {}"
	result := BuildReviewPrompt(profile, "You are a security expert.", query)

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

func TestBuildReviewPrompt_PersonaTrimmed(t *testing.T) {
	profile := provider.CodexRetryProfile() // 4000 char budget
	longPersona := strings.Repeat("expert ", 1000)
	query := "review this"

	result := BuildReviewPrompt(profile, longPersona, query)

	if len(result) > profile.ReviewBudgetChars+500 {
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

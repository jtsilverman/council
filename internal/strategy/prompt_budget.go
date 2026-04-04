package strategy

import (
	"fmt"
	"regexp"
	"strings"
	"unicode/utf8"

	"github.com/jtsilverman/council/internal/provider"
	"github.com/jtsilverman/council/internal/review"
)

var diffPathRe = regexp.MustCompile(`(?m)^diff --git a/(.+?) b/(.+?)$`)
var oldFileRe = regexp.MustCompile(`(?m)^--- a/(.+)$`)
var newFileRe = regexp.MustCompile(`(?m)^\+\+\+ b/(.+)$`)

const outputContract = `Respond in this exact format:

VERDICT: PASS | CONCERN | FAIL
FINDING: severity | file:line | title | rationale
FINDING: ...
SUMMARY: one-line summary

Severities: critical, high, medium, low. One VERDICT and SUMMARY required. Zero or more FINDINGs.`

// BuildReviewPrompt constructs a review prompt sized to the given profile.
func BuildReviewPrompt(profile provider.PromptProfile, persona, query string) string {
	var b strings.Builder

	if profile.RequireStructuredDigest {
		b.WriteString(outputContract)
		b.WriteString("\n\n")
	}

	q := query
	if profile.PreferWorkspaceRead {
		q = RewriteQueryForWorkspace(q)
	}

	// Trim persona if over budget (reserve space for query + contract)
	contractLen := b.Len()
	queryLen := len(q)
	personaBudget := profile.ReviewBudgetChars - contractLen - queryLen - 100 // 100 char margin
	if personaBudget < 0 {
		personaBudget = 0
	}
	trimmedPersona := persona
	if len(trimmedPersona) > personaBudget {
		if personaBudget > 0 {
			// Find the last valid rune boundary at or before personaBudget bytes
			for personaBudget > 0 && !utf8.RuneStart(trimmedPersona[personaBudget]) {
				personaBudget--
			}
			trimmedPersona = trimmedPersona[:personaBudget]
		} else {
			trimmedPersona = ""
		}
	}

	if trimmedPersona != "" {
		b.WriteString(trimmedPersona)
		b.WriteString("\n\n")
	}

	b.WriteString(q)
	return b.String()
}

// RewriteQueryForWorkspace replaces pasted file content with workspace-aware stubs.
// It extracts file paths from diff headers and removes full file contents,
// keeping only the diff itself plus a list of key files.
func RewriteQueryForWorkspace(query string) string {
	paths := extractDiffPaths(query)
	if len(paths) == 0 {
		return query
	}

	// Extract the diff portion (everything from the first "diff --git" line)
	diffStart := strings.Index(query, "diff --git")
	if diffStart < 0 {
		return query
	}

	diff := query[diffStart:]

	var b strings.Builder
	b.WriteString("Review the following diff. Read the full files from the workspace at the paths listed below if you need more context.\n\n")
	b.WriteString(diff)
	b.WriteString("\n\nKey files:\n")
	for _, p := range paths {
		fmt.Fprintf(&b, "- %s\n", p)
	}
	return b.String()
}

// extractDiffPaths pulls unique file paths from diff headers.
func extractDiffPaths(query string) []string {
	seen := make(map[string]bool)
	var paths []string

	for _, matches := range diffPathRe.FindAllStringSubmatch(query, -1) {
		if len(matches) >= 3 {
			for _, p := range matches[1:3] {
				if !seen[p] {
					seen[p] = true
					paths = append(paths, p)
				}
			}
		}
	}

	// Also check --- a/ and +++ b/ lines
	for _, matches := range oldFileRe.FindAllStringSubmatch(query, -1) {
		if len(matches) >= 2 && !seen[matches[1]] {
			seen[matches[1]] = true
			paths = append(paths, matches[1])
		}
	}
	for _, matches := range newFileRe.FindAllStringSubmatch(query, -1) {
		if len(matches) >= 2 && !seen[matches[1]] {
			seen[matches[1]] = true
			paths = append(paths, matches[1])
		}
	}

	return paths
}

// BuildDebatePrompt constructs a debate prompt with compacted phase-1 digests.
// The persona is passed separately as SystemPrompt in the Complete call, not here.
func BuildDebatePrompt(profile provider.PromptProfile, query string, digests []review.ReviewDigest) string {
	var b strings.Builder

	if profile.RequireStructuredDigest {
		b.WriteString(outputContract)
		b.WriteString("\n\n")
	}

	b.WriteString(fmt.Sprintf("Original query:\n%s\n\n", query))

	compacted := CompactDigests(digests, profile.DebateBudgetChars/2)
	b.WriteString("=== PHASE 1 FINDINGS ===\n\n")
	b.WriteString(compacted)
	b.WriteString("\n\n")

	b.WriteString(`Now review the other council members' findings above.

Your task:
1. CHALLENGE any finding you disagree with. State your specific technical reason.
2. SUPPORT any finding you think is especially important. Explain why.
3. ADD anything the other members missed that falls within your expertise.

Stay in character. Only challenge findings where you have specific technical grounds.
Do not repeat or summarize. Focus on disagreements and additions.`)

	return b.String()
}

// BuildSynthesisPrompt constructs a synthesis prompt with all compacted digests.
func BuildSynthesisPrompt(profile provider.PromptProfile, query string, allDigests []review.ReviewDigest) string {
	var b strings.Builder

	b.WriteString(fmt.Sprintf("Original query:\n%s\n\n", query))

	compacted := CompactDigests(allDigests, profile.SynthesisBudgetChars*3/4)
	b.WriteString("=== ALL FINDINGS ===\n\n")
	b.WriteString(compacted)
	b.WriteString("\n\n")

	b.WriteString(`=== YOUR TASK ===

Synthesize a final response based on the findings above.

Rules:
- Points with consensus support (2+ members agree) are high confidence. Include them.
- Points that were challenged with valid reasoning should be downgraded or dropped.
- Points that were supported during debate should be highlighted.
- Do not introduce new points that no member raised.
- Be decisive. Prioritize by impact.
- Produce a clear, well-organized final answer.`)

	return b.String()
}

// CompactDigests serializes digests within a character budget.
// If over budget: trims findings per member (keeping critical/high), truncates summaries,
// then drops lowest-severity findings.
func CompactDigests(digests []review.ReviewDigest, budgetChars int) string {
	if budgetChars <= 0 {
		budgetChars = 1000
	}

	// First pass: format all digests as-is
	parts := make([]string, len(digests))
	for i, d := range digests {
		parts[i] = review.FormatDigest(d)
	}
	result := strings.Join(parts, "\n")
	if len(result) <= budgetChars {
		return result
	}

	// Second pass: trim findings to 3 per member (keeps critical/high)
	trimmed := make([]review.ReviewDigest, len(digests))
	for i, d := range digests {
		trimmed[i] = review.TrimFindings(d, 3)
	}
	parts = make([]string, len(trimmed))
	for i, d := range trimmed {
		parts[i] = review.FormatDigest(d)
	}
	result = strings.Join(parts, "\n")
	if len(result) <= budgetChars {
		return result
	}

	// Third pass: trim findings to 2, truncate summaries
	for i, d := range digests {
		t := review.TrimFindings(d, 2)
		if len(t.Summary) > 100 {
			t.Summary = t.Summary[:100] + "..."
		}
		trimmed[i] = t
	}
	parts = make([]string, len(trimmed))
	for i, d := range trimmed {
		parts[i] = review.FormatDigest(d)
	}
	result = strings.Join(parts, "\n")
	if len(result) <= budgetChars {
		return result
	}

	// Fourth pass: only critical/high findings, minimal summaries
	for i, d := range digests {
		t := review.TrimFindings(d, 0) // only critical/high
		if len(t.Summary) > 50 {
			t.Summary = t.Summary[:50] + "..."
		}
		trimmed[i] = t
	}
	parts = make([]string, len(trimmed))
	for i, d := range trimmed {
		parts[i] = review.FormatDigest(d)
	}
	result = strings.Join(parts, "\n")

	// Final fallback: truncate at last newline before budget
	if len(result) > budgetChars {
		truncated := result[:budgetChars]
		if idx := strings.LastIndex(truncated, "\n"); idx > 0 {
			result = truncated[:idx]
		} else {
			result = truncated
		}
	}

	return result
}

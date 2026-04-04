package review

import (
	"strings"
	"testing"
)

func TestParseDigest_CleanStructured(t *testing.T) {
	raw := `VERDICT: PASS
FINDING: critical | main.go:42 | SQL injection | User input not sanitized
FINDING: low | utils.go:10 | Unused import | os package imported but not used
SUMMARY: Code is mostly clean with one critical security issue.`

	d := ParseDigest(raw)

	if d.Verdict != "PASS" {
		t.Errorf("verdict = %q, want PASS", d.Verdict)
	}
	if len(d.Findings) != 2 {
		t.Fatalf("findings = %d, want 2", len(d.Findings))
	}
	f0 := d.Findings[0]
	if f0.Severity != "critical" {
		t.Errorf("f0.Severity = %q, want critical", f0.Severity)
	}
	if f0.File != "main.go" {
		t.Errorf("f0.File = %q, want main.go", f0.File)
	}
	if f0.Line != "42" {
		t.Errorf("f0.Line = %q, want 42", f0.Line)
	}
	if f0.Title != "SQL injection" {
		t.Errorf("f0.Title = %q, want SQL injection", f0.Title)
	}
	if f0.Rationale != "User input not sanitized" {
		t.Errorf("f0.Rationale = %q, want 'User input not sanitized'", f0.Rationale)
	}
	if d.Summary != "Code is mostly clean with one critical security issue." {
		t.Errorf("summary = %q", d.Summary)
	}
}

func TestParseDigest_MessyMarkdown(t *testing.T) {
	raw := `### Verdict: FAIL

1. [severity: high] Missing error handling in database calls
2. [severity: low] Variable naming inconsistent

### Summary
Overall the code needs significant rework.`

	d := ParseDigest(raw)

	if d.Verdict != "FAIL" {
		t.Errorf("verdict = %q, want FAIL", d.Verdict)
	}
	if len(d.Findings) != 2 {
		t.Fatalf("findings = %d, want 2", len(d.Findings))
	}
	f0 := d.Findings[0]
	if f0.Severity != "high" {
		t.Errorf("f0.Severity = %q, want high", f0.Severity)
	}
	if f0.Title != "Missing error handling in database calls" {
		t.Errorf("f0.Title = %q", f0.Title)
	}
	if f0.File != "" {
		t.Errorf("f0.File = %q, want empty", f0.File)
	}
	if d.Summary != "Overall the code needs significant rework." {
		t.Errorf("summary = %q", d.Summary)
	}
}

func TestParseDigest_Unparseable(t *testing.T) {
	raw := "This is just random text that doesn't follow any format at all."

	d := ParseDigest(raw)

	if d.Verdict != "unknown" {
		t.Errorf("verdict = %q, want unknown", d.Verdict)
	}
	if len(d.Findings) != 1 {
		t.Fatalf("findings = %d, want 1", len(d.Findings))
	}
	if d.Findings[0].Severity != "unknown" {
		t.Errorf("severity = %q, want unknown", d.Findings[0].Severity)
	}
	if d.Findings[0].Title != raw {
		t.Errorf("title = %q, want raw text", d.Findings[0].Title)
	}
}

func TestParseDigest_UnparseableLong(t *testing.T) {
	raw := strings.Repeat("x", 1000)
	d := ParseDigest(raw)

	if len(d.Findings[0].Title) != 500 {
		t.Errorf("title length = %d, want 500", len(d.Findings[0].Title))
	}
}

func TestParseDigest_MissingFields(t *testing.T) {
	// Severity only, no file/line via numbered format
	raw := `VERDICT: CONCERN
1. [severity: medium] Some issue without file reference`

	d := ParseDigest(raw)

	if d.Verdict != "CONCERN" {
		t.Errorf("verdict = %q, want CONCERN", d.Verdict)
	}
	if len(d.Findings) != 1 {
		t.Fatalf("findings = %d, want 1", len(d.Findings))
	}
	f := d.Findings[0]
	if f.Severity != "medium" {
		t.Errorf("severity = %q, want medium", f.Severity)
	}
	if f.File != "" {
		t.Errorf("file = %q, want empty", f.File)
	}
	if f.Line != "" {
		t.Errorf("line = %q, want empty", f.Line)
	}
}

func TestParseDigest_MultilineRationale(t *testing.T) {
	raw := `VERDICT: FAIL
FINDING: high | server.go:99 | Memory leak | Goroutine not cleaned up
This happens because the channel is never closed.
The fix is to add a defer close() call.
SUMMARY: Needs fix before merge.`

	d := ParseDigest(raw)

	if len(d.Findings) != 1 {
		t.Fatalf("findings = %d, want 1", len(d.Findings))
	}
	f := d.Findings[0]
	expected := "Goroutine not cleaned up\nThis happens because the channel is never closed.\nThe fix is to add a defer close() call."
	if f.Rationale != expected {
		t.Errorf("rationale = %q, want %q", f.Rationale, expected)
	}
}

func TestParseDigest_DebateOutputSameAsReview(t *testing.T) {
	// Debate output uses same structured format
	debate := `VERDICT: PASS
FINDING: medium | api.go:15 | Missing rate limit | No throttling on public endpoint
SUMMARY: Acceptable for MVP but rate limiting needed before production.`

	review := `VERDICT: PASS
FINDING: medium | api.go:15 | Missing rate limit | No throttling on public endpoint
SUMMARY: Acceptable for MVP but rate limiting needed before production.`

	dd := ParseDigest(debate)
	rd := ParseDigest(review)

	if dd.Verdict != rd.Verdict {
		t.Errorf("verdicts differ: %q vs %q", dd.Verdict, rd.Verdict)
	}
	if len(dd.Findings) != len(rd.Findings) {
		t.Fatalf("finding counts differ: %d vs %d", len(dd.Findings), len(rd.Findings))
	}
	if dd.Findings[0].Title != rd.Findings[0].Title {
		t.Errorf("titles differ")
	}
	if dd.Summary != rd.Summary {
		t.Errorf("summaries differ")
	}
}

func TestTrimFindings_MixedSeverities(t *testing.T) {
	d := ReviewDigest{
		Verdict: "FAIL",
		Findings: []FindingNote{
			{Severity: "critical", Title: "SQL injection"},
			{Severity: "high", Title: "Auth bypass"},
			{Severity: "medium", Title: "Missing docs"},
			{Severity: "low", Title: "Typo in comment"},
			{Severity: "low", Title: "Unused variable"},
		},
		Summary: "Multiple issues found.",
	}

	// Trim to 3: must keep critical + high (2), plus 1 medium
	trimmed := TrimFindings(d, 3)

	if len(trimmed.Findings) != 3 {
		t.Fatalf("findings = %d, want 3", len(trimmed.Findings))
	}

	sevs := make(map[string]int)
	for _, f := range trimmed.Findings {
		sevs[f.Severity]++
	}
	if sevs["critical"] != 1 {
		t.Errorf("critical = %d, want 1", sevs["critical"])
	}
	if sevs["high"] != 1 {
		t.Errorf("high = %d, want 1", sevs["high"])
	}
	if sevs["medium"] != 1 {
		t.Errorf("medium = %d, want 1", sevs["medium"])
	}
}

func TestTrimFindings_RetainAllCriticalHigh(t *testing.T) {
	d := ReviewDigest{
		Verdict: "FAIL",
		Findings: []FindingNote{
			{Severity: "critical", Title: "Issue 1"},
			{Severity: "critical", Title: "Issue 2"},
			{Severity: "high", Title: "Issue 3"},
			{Severity: "high", Title: "Issue 4"},
			{Severity: "low", Title: "Issue 5"},
		},
	}

	// N=2 but 4 critical/high exist, all must be kept
	trimmed := TrimFindings(d, 2)

	if len(trimmed.Findings) != 4 {
		t.Fatalf("findings = %d, want 4 (all critical+high)", len(trimmed.Findings))
	}
}

func TestTrimFindings_PreservesMetadata(t *testing.T) {
	d := ReviewDigest{
		Verdict: "CONCERN",
		Summary: "Some summary",
		Findings: []FindingNote{
			{Severity: "low", Title: "A"},
		},
	}
	trimmed := TrimFindings(d, 1)
	if trimmed.Verdict != "CONCERN" {
		t.Errorf("verdict = %q", trimmed.Verdict)
	}
	if trimmed.Summary != "Some summary" {
		t.Errorf("summary = %q", trimmed.Summary)
	}
}

func TestFormatDigest(t *testing.T) {
	d := ReviewDigest{
		Verdict: "PASS",
		Findings: []FindingNote{
			{Severity: "high", File: "main.go", Line: "10", Title: "Bug", Rationale: "Crash on nil"},
		},
		Summary: "Looks good overall.",
	}

	out := FormatDigest(d)

	if !strings.Contains(out, "VERDICT: PASS") {
		t.Errorf("missing VERDICT line")
	}
	if !strings.Contains(out, "FINDING: high | main.go:10 | Bug | Crash on nil") {
		t.Errorf("missing FINDING line, got: %s", out)
	}
	if !strings.Contains(out, "SUMMARY: Looks good overall.") {
		t.Errorf("missing SUMMARY line")
	}
}

func TestFormatDigest_RoundTrip(t *testing.T) {
	original := ReviewDigest{
		Verdict: "FAIL",
		Findings: []FindingNote{
			{Severity: "critical", File: "db.go", Line: "55", Title: "Data loss", Rationale: "Transaction not committed"},
			{Severity: "low", File: "fmt.go", Line: "3", Title: "Style", Rationale: "Use gofmt"},
		},
		Summary: "Critical data loss bug.",
	}

	serialized := FormatDigest(original)
	reparsed := ParseDigest(serialized)

	if reparsed.Verdict != original.Verdict {
		t.Errorf("verdict: %q vs %q", reparsed.Verdict, original.Verdict)
	}
	if len(reparsed.Findings) != len(original.Findings) {
		t.Fatalf("findings: %d vs %d", len(reparsed.Findings), len(original.Findings))
	}
	for i, f := range reparsed.Findings {
		o := original.Findings[i]
		if f.Severity != o.Severity || f.File != o.File || f.Line != o.Line || f.Title != o.Title || f.Rationale != o.Rationale {
			t.Errorf("finding %d mismatch: %+v vs %+v", i, f, o)
		}
	}
	if reparsed.Summary != original.Summary {
		t.Errorf("summary: %q vs %q", reparsed.Summary, original.Summary)
	}
}

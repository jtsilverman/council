package review

import (
	"fmt"
	"strings"
)

// ReviewDigest holds parsed structured review output.
type ReviewDigest struct {
	Verdict  string
	Findings []FindingNote
	Summary  string
}

// FindingNote represents a single finding from a review.
type FindingNote struct {
	Severity  string // critical, high, medium, low
	File      string // file:line reference
	Line      string
	Title     string
	Rationale string
}

// ParseDigest parses raw review or debate output into a ReviewDigest.
// It handles both structured (VERDICT/FINDING/SUMMARY) and markdown formats.
func ParseDigest(raw string) ReviewDigest {
	lines := strings.Split(raw, "\n")
	d := ReviewDigest{}

	type section int
	const (
		sNone section = iota
		sFinding
		sSummary
	)

	cur := sNone
	var findingIdx int

	for i := 0; i < len(lines); i++ {
		line := lines[i]
		trimmed := strings.TrimSpace(line)

		// VERDICT: PASS|CONCERN|FAIL
		if v, ok := parseVerdict(trimmed); ok {
			d.Verdict = v
			cur = sNone
			continue
		}

		// FINDING: severity | file:line | title | rationale
		if f, ok := parseFindingLine(trimmed); ok {
			d.Findings = append(d.Findings, f)
			findingIdx = len(d.Findings) - 1
			cur = sFinding
			continue
		}

		// Numbered list format: 1. [severity: X] description
		if f, ok := parseNumberedFinding(trimmed); ok {
			d.Findings = append(d.Findings, f)
			findingIdx = len(d.Findings) - 1
			cur = sFinding
			continue
		}

		// SUMMARY: text
		if s, ok := parseSummaryLine(trimmed); ok {
			d.Summary = s
			cur = sSummary
			continue
		}

		// Multiline continuation
		if trimmed == "" {
			continue
		}
		switch cur {
		case sFinding:
			// Append to rationale of current finding
			if d.Findings[findingIdx].Rationale == "" {
				d.Findings[findingIdx].Rationale = trimmed
			} else {
				d.Findings[findingIdx].Rationale += "\n" + trimmed
			}
		case sSummary:
			if d.Summary == "" {
				d.Summary = trimmed
			} else {
				d.Summary += "\n" + trimmed
			}
		}
	}

	// Fallback: if no verdict and no findings found, produce fallback
	if d.Verdict == "" && len(d.Findings) == 0 {
		title := raw
		if len(title) > 500 {
			title = title[:500]
		}
		d.Verdict = "unknown"
		d.Findings = []FindingNote{{
			Severity: "unknown",
			Title:    title,
		}}
	} else if d.Verdict == "" {
		d.Verdict = "unknown"
	}

	return d
}

func parseVerdict(line string) (string, bool) {
	// VERDICT: PASS
	upper := strings.ToUpper(line)
	if strings.HasPrefix(upper, "VERDICT:") {
		v := strings.TrimSpace(line[len("VERDICT:"):])
		return normalizeVerdict(v), true
	}
	// ### Verdict: PASS
	stripped := strings.TrimLeft(line, "# ")
	upperStripped := strings.ToUpper(stripped)
	if strings.HasPrefix(upperStripped, "VERDICT:") {
		v := strings.TrimSpace(stripped[len("VERDICT:"):])
		return normalizeVerdict(v), true
	}
	return "", false
}

func normalizeVerdict(v string) string {
	upper := strings.ToUpper(strings.TrimSpace(v))
	switch upper {
	case "PASS":
		return "PASS"
	case "FAIL":
		return "FAIL"
	case "CONCERN":
		return "CONCERN"
	default:
		return strings.TrimSpace(v)
	}
}

func parseFindingLine(line string) (FindingNote, bool) {
	upper := strings.ToUpper(line)
	if !strings.HasPrefix(upper, "FINDING:") {
		return FindingNote{}, false
	}
	rest := strings.TrimSpace(line[len("FINDING:"):])
	parts := strings.SplitN(rest, "|", 4)

	f := FindingNote{}
	if len(parts) >= 1 {
		f.Severity = strings.ToLower(strings.TrimSpace(parts[0]))
	}
	if len(parts) >= 2 {
		fileRef := strings.TrimSpace(parts[1])
		if idx := strings.LastIndex(fileRef, ":"); idx > 0 {
			f.File = fileRef[:idx]
			f.Line = fileRef[idx+1:]
		} else {
			f.File = fileRef
		}
	}
	if len(parts) >= 3 {
		f.Title = strings.TrimSpace(parts[2])
	}
	if len(parts) >= 4 {
		f.Rationale = strings.TrimSpace(parts[3])
	}
	return f, true
}

func parseNumberedFinding(line string) (FindingNote, bool) {
	// Match: 1. [severity: X] description
	// or: - [severity: X] description
	rest := line
	if len(rest) == 0 {
		return FindingNote{}, false
	}

	// Strip leading number + dot or dash
	if rest[0] == '-' {
		rest = strings.TrimSpace(rest[1:])
	} else if rest[0] >= '0' && rest[0] <= '9' {
		dotIdx := strings.Index(rest, ".")
		if dotIdx < 0 || dotIdx > 5 {
			return FindingNote{}, false
		}
		rest = strings.TrimSpace(rest[dotIdx+1:])
	} else {
		return FindingNote{}, false
	}

	// Must start with [severity: ...]
	if !strings.HasPrefix(rest, "[") {
		return FindingNote{}, false
	}
	closeBracket := strings.Index(rest, "]")
	if closeBracket < 0 {
		return FindingNote{}, false
	}
	bracketContent := rest[1:closeBracket]
	description := strings.TrimSpace(rest[closeBracket+1:])

	// Parse severity from bracket content
	severity := bracketContent
	if idx := strings.Index(strings.ToLower(bracketContent), "severity:"); idx >= 0 {
		severity = strings.TrimSpace(bracketContent[idx+len("severity:"):])
	}
	severity = strings.ToLower(strings.TrimSpace(severity))

	return FindingNote{
		Severity: severity,
		Title:    description,
	}, true
}

func parseSummaryLine(line string) (string, bool) {
	upper := strings.ToUpper(line)
	if strings.HasPrefix(upper, "SUMMARY:") {
		return strings.TrimSpace(line[len("SUMMARY:"):]), true
	}
	// ### Summary
	stripped := strings.TrimLeft(line, "# ")
	upperStripped := strings.ToUpper(stripped)
	if strings.HasPrefix(upperStripped, "SUMMARY:") {
		return strings.TrimSpace(stripped[len("SUMMARY:"):]), true
	}
	if strings.ToUpper(strings.TrimSpace(stripped)) == "SUMMARY" {
		return "", true
	}
	return "", false
}

// TrimFindings keeps the top N findings but always retains all critical and high severity.
func TrimFindings(d ReviewDigest, n int) ReviewDigest {
	result := ReviewDigest{
		Verdict: d.Verdict,
		Summary: d.Summary,
	}

	var kept []FindingNote
	var rest []FindingNote

	for _, f := range d.Findings {
		sev := strings.ToLower(f.Severity)
		if sev == "critical" || sev == "high" {
			kept = append(kept, f)
		} else {
			rest = append(rest, f)
		}
	}

	remaining := n - len(kept)
	if remaining > 0 && len(rest) > 0 {
		if remaining > len(rest) {
			remaining = len(rest)
		}
		kept = append(kept, rest[:remaining]...)
	}

	result.Findings = kept
	return result
}

// FormatDigest re-serializes a ReviewDigest to structured format.
func FormatDigest(d ReviewDigest) string {
	var b strings.Builder

	fmt.Fprintf(&b, "VERDICT: %s\n", d.Verdict)

	for _, f := range d.Findings {
		fileRef := f.File
		if f.Line != "" {
			fileRef += ":" + f.Line
		}
		fmt.Fprintf(&b, "FINDING: %s | %s | %s | %s\n", f.Severity, fileRef, f.Title, f.Rationale)
	}

	fmt.Fprintf(&b, "SUMMARY: %s\n", d.Summary)

	return b.String()
}

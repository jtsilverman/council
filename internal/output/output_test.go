package output

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/jtsilverman/council/internal/council"
)

func TestSanitize_ANSIColorCodes(t *testing.T) {
	input := "\033[31mred text\033[0m normal"
	got := sanitize(input)
	want := "red text normal"
	if got != want {
		t.Errorf("sanitize() = %q, want %q", got, want)
	}
}

func TestSanitize_NoANSI(t *testing.T) {
	input := "plain text with no escape codes"
	got := sanitize(input)
	if got != input {
		t.Errorf("sanitize() = %q, want %q (passthrough)", got, input)
	}
}

func TestSanitize_CursorMovement(t *testing.T) {
	input := "\033[2Amove up\033[10Bmove down\033[5Cforward"
	got := sanitize(input)
	want := "move upmove downforward"
	if got != want {
		t.Errorf("sanitize() = %q, want %q", got, want)
	}
}

func TestSanitize_Empty(t *testing.T) {
	got := sanitize("")
	if got != "" {
		t.Errorf("sanitize() = %q, want empty", got)
	}
}

func TestSanitize_OSCSequences(t *testing.T) {
	// OSC (Operating System Command) sequences: ESC ] ... BEL
	input := "\033]0;window title\007rest of text"
	got := sanitize(input)
	want := "rest of text"
	if got != want {
		t.Errorf("sanitize() = %q, want %q", got, want)
	}
}

func TestSanitize_MultipleSequences(t *testing.T) {
	input := "\033[1m\033[32mbold green\033[0m \033[4munderline\033[0m"
	got := sanitize(input)
	want := "bold green underline"
	if got != want {
		t.Errorf("sanitize() = %q, want %q", got, want)
	}
}

func newTestDeliberation() *council.Deliberation {
	return &council.Deliberation{
		Query:    "review this code",
		Council:  "test-council",
		Strategy: "debate",
		Rounds: []council.Round{
			{
				Phase: "review",
				Responses: []council.Response{
					{
						Member:  "Alice",
						Content: "Looks good.",
						Tokens:  council.TokenUsage{Input: 100, Output: 50, Cost: 0.001},
						Latency: 500 * time.Millisecond,
					},
					{
						Member:  "Bob",
						Content: "Found a \033[31mbug\033[0m.",
						Tokens:  council.TokenUsage{Input: 100, Output: 60, Cost: 0.002},
						Latency: 600 * time.Millisecond,
					},
				},
			},
		},
		Synthesis: council.Response{
			Member:  "Chair",
			Content: "One minor bug found.",
			Tokens:  council.TokenUsage{Input: 200, Output: 80, Cost: 0.003},
			Latency: 400 * time.Millisecond,
		},
		TotalTokens: council.TokenUsage{Input: 400, Output: 190, Cost: 0.006},
		TotalCost:   0.006,
		Duration:    2 * time.Second,
	}
}

func TestRenderJSON_ValidOutput(t *testing.T) {
	d := newTestDeliberation()
	var buf bytes.Buffer

	if err := RenderJSON(&buf, d); err != nil {
		t.Fatalf("RenderJSON() error: %v", err)
	}

	// Output must be valid JSON
	var parsed map[string]interface{}
	if err := json.Unmarshal(buf.Bytes(), &parsed); err != nil {
		t.Fatalf("output is not valid JSON: %v\nOutput: %s", err, buf.String())
	}

	if parsed["council"] != "test-council" {
		t.Errorf("council = %v, want test-council", parsed["council"])
	}
	if parsed["query"] != "review this code" {
		t.Errorf("query = %v", parsed["query"])
	}
}

func TestRenderJSON_EmptyDeliberation(t *testing.T) {
	d := &council.Deliberation{}
	var buf bytes.Buffer

	if err := RenderJSON(&buf, d); err != nil {
		t.Fatalf("RenderJSON() error: %v", err)
	}

	var parsed map[string]interface{}
	if err := json.Unmarshal(buf.Bytes(), &parsed); err != nil {
		t.Fatalf("output is not valid JSON: %v", err)
	}
}

func TestRenderTerminal_ConciseNoPanic(t *testing.T) {
	d := newTestDeliberation()
	var buf bytes.Buffer

	if err := RenderTerminal(&buf, d, false); err != nil {
		t.Fatalf("RenderTerminal() error: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "Final Answer") {
		t.Error("concise output should contain 'Final Answer'")
	}
	if !strings.Contains(out, "One minor bug found.") {
		t.Error("concise output should contain synthesis content")
	}
	if !strings.Contains(out, "test-council") {
		t.Error("concise output should contain council name")
	}
}

func TestRenderTerminal_VerboseNoPanic(t *testing.T) {
	d := newTestDeliberation()
	var buf bytes.Buffer

	if err := RenderTerminal(&buf, d, true); err != nil {
		t.Fatalf("RenderTerminal() error: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "REVIEW") {
		t.Error("verbose output should contain phase name")
	}
	if !strings.Contains(out, "Alice") {
		t.Error("verbose output should contain member names")
	}
	if !strings.Contains(out, "SYNTHESIS") {
		t.Error("verbose output should contain SYNTHESIS header")
	}
}

func TestRenderTerminal_SanitizesANSI(t *testing.T) {
	d := newTestDeliberation()
	var buf bytes.Buffer

	if err := RenderTerminal(&buf, d, true); err != nil {
		t.Fatalf("RenderTerminal() error: %v", err)
	}

	out := buf.String()
	// The ANSI codes in Bob's response ("\033[31mbug\033[0m") should be stripped
	// from the member content. The output itself uses ANSI for formatting, so
	// just check that the raw red escape from the member content is gone.
	if strings.Contains(out, "\033[31mbug") {
		t.Error("member ANSI codes should be sanitized from content")
	}
}

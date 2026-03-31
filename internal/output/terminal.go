package output

import (
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/jtsilverman/council/internal/council"
)

// Colors for terminal output (ANSI escape codes).
const (
	colorReset  = "\033[0m"
	colorBold   = "\033[1m"
	colorDim    = "\033[2m"
	colorGreen  = "\033[32m"
	colorYellow = "\033[33m"
	colorBlue   = "\033[34m"
	colorCyan   = "\033[36m"
	colorWhite  = "\033[37m"
)

// RenderTerminal renders a deliberation to the terminal.
func RenderTerminal(w io.Writer, d *council.Deliberation, verbose bool) error {
	if verbose {
		return renderVerbose(w, d)
	}
	return renderConcise(w, d)
}

func renderConcise(w io.Writer, d *council.Deliberation) error {
	// Show the synthesis (final answer)
	fmt.Fprintf(w, "\n%s%s Council — Final Answer%s\n", colorBold, colorGreen, colorReset)
	fmt.Fprintf(w, "%s%s(synthesized by %s)%s\n\n", colorDim, colorWhite, d.Synthesis.Member, colorReset)
	fmt.Fprintln(w, d.Synthesis.Content)

	// Footer with stats
	fmt.Fprintf(w, "\n%s%s─────────────────────────────────%s\n", colorDim, colorWhite, colorReset)
	memberCount := 0
	if len(d.Rounds) > 0 {
		memberCount = len(d.Rounds[0].Responses)
	}
	fmt.Fprintf(w, "%s%sCouncil: %s | Members: %d | Strategy: %s | Duration: %s%s\n",
		colorDim, colorWhite,
		d.Council, memberCount, d.Strategy,
		formatDuration(d.Duration), colorReset)
	if d.TotalCost > 0 {
		fmt.Fprintf(w, "%s%sTokens: %d in / %d out | Cost: $%.4f%s\n",
			colorDim, colorWhite,
			d.TotalTokens.Input, d.TotalTokens.Output, d.TotalCost, colorReset)
	}
	return nil
}

func renderVerbose(w io.Writer, d *council.Deliberation) error {
	for _, round := range d.Rounds {
		phase := strings.ToUpper(round.Phase)
		fmt.Fprintf(w, "\n%s%s═══ %s ═══%s\n", colorBold, colorYellow, phase, colorReset)

		for _, r := range round.Responses {
			fmt.Fprintf(w, "\n%s%s── %s%s", colorBold, colorCyan, r.Member, colorReset)
			if r.Latency > 0 {
				fmt.Fprintf(w, " %s%s(%s)%s", colorDim, colorWhite, formatDuration(r.Latency), colorReset)
			}
			fmt.Fprintln(w)
			fmt.Fprintln(w, r.Content)
		}
	}

	fmt.Fprintf(w, "\n%s%s═══ SYNTHESIS ═══%s\n", colorBold, colorGreen, colorReset)
	fmt.Fprintf(w, "%s%s── %s%s\n", colorBold, colorGreen, d.Synthesis.Member, colorReset)
	fmt.Fprintln(w, d.Synthesis.Content)

	// Footer
	fmt.Fprintf(w, "\n%s%s─────────────────────────────────%s\n", colorDim, colorWhite, colorReset)
	memberCount := 0
	if len(d.Rounds) > 0 {
		memberCount = len(d.Rounds[0].Responses)
	}
	fmt.Fprintf(w, "%s%sCouncil: %s | Members: %d | Strategy: %s | Duration: %s%s\n",
		colorDim, colorWhite,
		d.Council, memberCount, d.Strategy,
		formatDuration(d.Duration), colorReset)
	if d.TotalCost > 0 {
		fmt.Fprintf(w, "%s%sTokens: %d in / %d out | Cost: $%.4f%s\n",
			colorDim, colorWhite,
			d.TotalTokens.Input, d.TotalTokens.Output, d.TotalCost, colorReset)
	}
	return nil
}

func formatDuration(d time.Duration) string {
	if d < time.Second {
		return fmt.Sprintf("%dms", d.Milliseconds())
	}
	return fmt.Sprintf("%.1fs", d.Seconds())
}

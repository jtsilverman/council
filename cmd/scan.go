package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/spf13/cobra"

	"github.com/jtsilverman/council/internal/council"
	"github.com/jtsilverman/council/internal/output"
	"github.com/jtsilverman/council/internal/persona"
	"github.com/jtsilverman/council/internal/provider"
	"github.com/jtsilverman/council/internal/strategy"
)

// Source file extensions to review.
var sourceExts = map[string]bool{
	".go":   true,
	".py":   true,
	".js":   true,
	".ts":   true,
	".tsx":  true,
	".jsx":  true,
	".rs":   true,
	".java": true,
	".rb":   true,
	".c":    true,
	".cpp":  true,
	".h":    true,
	".cs":   true,
	".php":  true,
	".swift": true,
	".kt":  true,
}

// Directories to skip.
var skipDirs = map[string]bool{
	"node_modules": true, "vendor": true, ".git": true,
	"dist": true, "build": true, "target": true,
	"__pycache__": true, ".venv": true, "venv": true,
	".next": true, ".cache": true, "coverage": true,
}

var scanCmd = &cobra.Command{
	Use:   "scan <dir>",
	Short: "Review all source files in a directory",
	Long: `Scan walks a directory, finds source files, and runs the code-review
council on each file. Uses vote strategy (light, fast) by default.
Files with critical findings are flagged for deep review.`,
	Args: cobra.ExactArgs(1),
	RunE: runScan,
}

func init() {
	rootCmd.AddCommand(scanCmd)
}

type scanResult struct {
	File     string
	Delib    *council.Deliberation
	Critical bool
}

func runScan(cmd *cobra.Command, args []string) error {
	dir := args[0]

	// Find source files
	files, err := findSourceFiles(dir)
	if err != nil {
		return fmt.Errorf("scan directory: %w", err)
	}
	if len(files) == 0 {
		return fmt.Errorf("no source files found in %s", dir)
	}

	fmt.Fprintf(os.Stderr, "Found %d source files in %s\n", len(files), dir)

	// Get council
	c, err := persona.GetCouncil("code-review")
	if err != nil {
		return err
	}

	// Light review: 2 members, vote strategy
	scanStrategy := "vote"
	scanMembers := 2
	if flagStrategy != "" {
		scanStrategy = flagStrategy
	}
	if flagMembers > 0 {
		scanMembers = flagMembers
	}

	c.Strategy = scanStrategy
	if scanMembers < len(c.Members) {
		c.Members = c.Members[:scanMembers]
	}

	if flagModel != "" {
		for i := range c.Members {
			c.Members[i].Model = flagModel
		}
		c.Chair.Model = flagModel
	}

	// Create provider
	var p provider.Provider
	if flagAPI {
		p, err = provider.NewAnthropicProvider()
		if err != nil {
			return err
		}
	} else {
		p = provider.NewCLIProvider(flagModel)
	}

	strat := strategy.Get(c.Strategy)

	// Scan files with concurrency limit
	fmt.Fprintf(os.Stderr, "Reviewing with %d members, %s strategy...\n\n", len(c.Members), c.Strategy)

	var results []scanResult
	var mu sync.Mutex
	var wg sync.WaitGroup
	sem := make(chan struct{}, 2) // 2 files at a time to avoid overwhelming CLI
	var completed int64

	start := time.Now()

	for _, file := range files {
		wg.Add(1)
		go func(f string) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			content, err := os.ReadFile(f)
			if err != nil {
				fmt.Fprintf(os.Stderr, "  Skip %s: %v\n", f, err)
				return
			}

			// Skip very large files (>50KB) and empty files
			if len(content) == 0 || len(content) > 50000 {
				n := atomic.AddInt64(&completed, 1)
				fmt.Fprintf(os.Stderr, "  [%d/%d] Skip %s (size: %d bytes)\n", n, len(files), relPath(dir, f), len(content))
				return
			}

			// Build query with file context
			relFile := relPath(dir, f)
			query := fmt.Sprintf("Review this file: %s\n\n```\n%s\n```", relFile, string(content))

			// Make a copy of council for this file
			fileCopy := *c
			members := make([]council.Member, len(c.Members))
			copy(members, c.Members)
			fileCopy.Members = members

			delib, err := council.Run(cmd.Context(), &fileCopy, query, p, strat)
			if err != nil {
				n := atomic.AddInt64(&completed, 1)
				fmt.Fprintf(os.Stderr, "  [%d/%d] Error %s: %v\n", n, len(files), relFile, err)
				return
			}

			// Check if synthesis mentions "critical" or "security"
			synthLower := strings.ToLower(delib.Synthesis.Content)
			critical := strings.Contains(synthLower, "critical") ||
				strings.Contains(synthLower, "vulnerability") ||
				strings.Contains(synthLower, "injection") ||
				strings.Contains(synthLower, "security flaw")

			n := atomic.AddInt64(&completed, 1)
			status := "✓"
			if critical {
				status = "⚠"
			}
			fmt.Fprintf(os.Stderr, "  [%d/%d] %s %s\n", n, len(files), status, relFile)

			mu.Lock()
			results = append(results, scanResult{
				File:     relFile,
				Delib:    delib,
				Critical: critical,
			})
			mu.Unlock()
		}(file)
	}

	wg.Wait()
	elapsed := time.Since(start)

	// Output results
	if flagJSON {
		return renderScanJSON(results, dir, elapsed)
	}
	return renderScanTerminal(results, dir, elapsed)
}

func renderScanTerminal(results []scanResult, dir string, elapsed time.Duration) error {
	// Summary
	criticalCount := 0
	for _, r := range results {
		if r.Critical {
			criticalCount++
		}
	}

	fmt.Printf("\n\033[1m\033[32m═══ Scan Results: %s ═══\033[0m\n", dir)
	fmt.Printf("\033[2m\033[37mFiles reviewed: %d | Critical: %d | Duration: %s\033[0m\n\n", len(results), criticalCount, formatScanDuration(elapsed))

	// Show critical files first
	if criticalCount > 0 {
		fmt.Printf("\033[1m\033[31m── Critical Findings ──\033[0m\n\n")
		for _, r := range results {
			if r.Critical {
				fmt.Printf("\033[1m\033[31m⚠ %s\033[0m\n", r.File)
				fmt.Println(r.Delib.Synthesis.Content)
				fmt.Println()
			}
		}
	}

	// Show non-critical files
	fmt.Printf("\033[1m\033[32m── Other Files ──\033[0m\n\n")
	for _, r := range results {
		if !r.Critical {
			fmt.Printf("\033[1m\033[36m✓ %s\033[0m\n", r.File)
			// Show just a brief summary (first 3 lines of synthesis)
			lines := strings.SplitN(r.Delib.Synthesis.Content, "\n", 4)
			for _, l := range lines[:min(len(lines), 3)] {
				if strings.TrimSpace(l) != "" {
					fmt.Printf("  %s\n", l)
				}
			}
			fmt.Println()
		}
	}

	return nil
}

func findSourceFiles(dir string) ([]string, error) {
	var files []string
	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if info.IsDir() {
			if skipDirs[info.Name()] {
				return filepath.SkipDir
			}
			return nil
		}
		ext := strings.ToLower(filepath.Ext(info.Name()))
		if sourceExts[ext] {
			files = append(files, path)
		}
		return nil
	})
	return files, err
}

func relPath(base, path string) string {
	rel, err := filepath.Rel(base, path)
	if err != nil {
		return path
	}
	return rel
}

func formatScanDuration(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%.0fs", d.Seconds())
	}
	return fmt.Sprintf("%dm%02ds", int(d.Minutes()), int(d.Seconds())%60)
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func renderScanJSON(results []scanResult, dir string, elapsed time.Duration) error {
	type jsonFileResult struct {
		File      string `json:"file"`
		Critical  bool   `json:"critical"`
		Synthesis string `json:"synthesis"`
	}
	type jsonReport struct {
		Directory string           `json:"directory"`
		Files     int              `json:"files_reviewed"`
		Critical  int              `json:"critical_count"`
		Duration  string           `json:"duration"`
		Results   []jsonFileResult `json:"results"`
	}

	critCount := 0
	var fileResults []jsonFileResult
	for _, r := range results {
		if r.Critical {
			critCount++
		}
		fileResults = append(fileResults, jsonFileResult{
			File:      r.File,
			Critical:  r.Critical,
			Synthesis: r.Delib.Synthesis.Content,
		})
	}

	return output.RenderAnyJSON(os.Stdout, jsonReport{
		Directory: dir,
		Files:     len(results),
		Critical:  critCount,
		Duration:  elapsed.String(),
		Results:   fileResults,
	})
}

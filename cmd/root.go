package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var (
	flagJSON     bool
	flagVerbose  bool
	flagModel    string
	flagModels   string
	flagCouncil  string
	flagStrategy string
	flagConfig   string
	flagDeep     bool
	flagLight    bool
	flagAll      bool
	flagWith     string
)

var rootCmd = &cobra.Command{
	Use:   "council [query]",
	Short: "Multi-perspective LLM deliberation",
	Long: `Council runs your query through multiple LLM expert personas in parallel,
then runs a debate round where they challenge each other, and a chair
synthesizes the final answer.

By default, uses claude --print (subscription, $0 cost).
Use --models to mix different LLM providers.`,
	Args: cobra.ArbitraryArgs,
	RunE: runCouncil,
}

func init() {
	rootCmd.PersistentFlags().BoolVar(&flagJSON, "json", false, "Output as JSON")
	rootCmd.PersistentFlags().BoolVar(&flagVerbose, "verbose", false, "Show deliberation details")
	rootCmd.PersistentFlags().StringVar(&flagModel, "model", "", "Override model for all members")
	rootCmd.PersistentFlags().StringVar(&flagModels, "models", "", "Comma-separated models (auto-detects providers)")
	rootCmd.PersistentFlags().StringVar(&flagCouncil, "council", "general", "Council to use (general, code-review, writing)")
	rootCmd.PersistentFlags().StringVar(&flagStrategy, "strategy", "", "Deliberation strategy (debate, vote)")
	rootCmd.PersistentFlags().StringVar(&flagConfig, "config", "", "Path to custom council YAML config")
	rootCmd.PersistentFlags().BoolVar(&flagDeep, "deep", false, "Full debate strategy (slower, more thorough)")
	rootCmd.PersistentFlags().BoolVar(&flagLight, "light", false, "Light review: 2 members only (Security + Bug Hunter)")
	rootCmd.PersistentFlags().BoolVar(&flagAll, "all", false, "All 10 code review members")
	rootCmd.PersistentFlags().StringVar(&flagWith, "with", "", "Pick specific members by name (e.g. \"security,concurrency,data\")")
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

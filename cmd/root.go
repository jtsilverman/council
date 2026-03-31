package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var (
	flagJSON     bool
	flagAPI      bool
	flagVerbose  bool
	flagProvider string
	flagModel    string
	flagCouncil  string
	flagMembers  int
	flagStrategy string
)

var rootCmd = &cobra.Command{
	Use:   "council [query]",
	Short: "Multi-perspective LLM deliberation",
	Long: `Council runs your query through multiple LLM expert personas in parallel,
then runs a debate round where they challenge each other, and a chair
synthesizes the final answer.

By default, uses claude --print (subscription, $0 cost).
Use --api to use the Anthropic API instead.`,
	Args: cobra.ArbitraryArgs,
	RunE: runCouncil,
}

func init() {
	rootCmd.PersistentFlags().BoolVar(&flagJSON, "json", false, "Output as JSON")
	rootCmd.PersistentFlags().BoolVar(&flagAPI, "api", false, "Use API instead of CLI (costs money)")
	rootCmd.PersistentFlags().BoolVar(&flagVerbose, "verbose", false, "Show deliberation details")
	rootCmd.PersistentFlags().StringVar(&flagProvider, "provider", "anthropic", "LLM provider (anthropic, openai, gemini, ollama)")
	rootCmd.PersistentFlags().StringVar(&flagModel, "model", "", "Override model for all members")
	rootCmd.PersistentFlags().StringVar(&flagCouncil, "council", "general", "Council to use (general, code-review, writing)")
	rootCmd.PersistentFlags().IntVar(&flagMembers, "members", 0, "Override council size (2-7)")
	rootCmd.PersistentFlags().StringVar(&flagStrategy, "strategy", "", "Deliberation strategy (debate, vote, ranked)")
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

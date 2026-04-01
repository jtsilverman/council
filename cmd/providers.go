package cmd

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/jtsilverman/council/internal/provider"
)

var providersCmd = &cobra.Command{
	Use:   "providers",
	Short: "Show available LLM providers and models",
	RunE:  runProviders,
}

func init() {
	rootCmd.AddCommand(providersCmd)
}

func runProviders(cmd *cobra.Command, args []string) error {
	statuses := provider.DetectAll()

	fmt.Println("Detected providers:")
	fmt.Println()

	for _, s := range statuses {
		status := "\033[32m✓\033[0m"
		if !s.Available {
			status = "\033[31m✗\033[0m"
		}

		method := ""
		if s.Method != "" {
			method = fmt.Sprintf(" (%s)", s.Method)
		}

		fmt.Printf("  %s \033[1m%-12s\033[0m%s  %s\n", status, s.Name, method, s.Detail)
		if s.Available && len(s.Models) > 0 {
			fmt.Printf("    Models: %s\n", strings.Join(s.Models, ", "))
		}
	}

	fmt.Println()
	fmt.Println("Use --models to pick models: council --models \"claude-opus-4-6,gpt-5.4\" \"question\"")
	fmt.Println("Default (no flags): uses Claude via CLI subscription ($0)")

	return nil
}

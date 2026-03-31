package cmd

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/jtsilverman/council/internal/council"
	"github.com/jtsilverman/council/internal/output"
	"github.com/jtsilverman/council/internal/persona"
	"github.com/jtsilverman/council/internal/provider"
	"github.com/jtsilverman/council/internal/strategy"
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List available councils",
	RunE:  runList,
}

func init() {
	rootCmd.AddCommand(listCmd)
}

func runCouncil(cmd *cobra.Command, args []string) error {
	// Read query from args or stdin
	query := strings.Join(args, " ")
	if query == "" {
		stat, _ := os.Stdin.Stat()
		if (stat.Mode() & os.ModeCharDevice) == 0 {
			data, err := io.ReadAll(os.Stdin)
			if err != nil {
				return fmt.Errorf("read stdin: %w", err)
			}
			query = string(data)
		}
	}
	if strings.TrimSpace(query) == "" {
		return fmt.Errorf("no query provided. Usage: council \"your question\" or echo \"question\" | council")
	}

	// Look up council
	c, err := persona.GetCouncil(flagCouncil)
	if err != nil {
		return err
	}

	// Override strategy if specified
	if flagStrategy != "" {
		c.Strategy = flagStrategy
	}

	// Override model if specified
	if flagModel != "" {
		for i := range c.Members {
			c.Members[i].Model = flagModel
		}
		c.Chair.Model = flagModel
	}

	// Trim council size if specified
	if flagMembers > 0 && flagMembers < len(c.Members) {
		c.Members = c.Members[:flagMembers]
	}

	// Create provider
	var p provider.Provider
	if flagAPI {
		p, err = provider.NewAnthropicProvider()
		if err != nil {
			return fmt.Errorf("create API provider: %w", err)
		}
	} else {
		p = provider.NewCLIProvider(flagModel)
	}

	// Get strategy
	strat := strategy.Get(c.Strategy)

	// Run deliberation
	fmt.Fprintf(os.Stderr, "Running %s council (%d members, %s strategy)...\n", c.Name, len(c.Members), c.Strategy)
	delib, err := council.Run(cmd.Context(), c, query, p, strat)
	if err != nil {
		return fmt.Errorf("council deliberation failed: %w", err)
	}

	// Output
	if flagJSON {
		return output.RenderJSON(os.Stdout, delib)
	}
	return output.RenderTerminal(os.Stdout, delib, flagVerbose)
}

func runList(cmd *cobra.Command, args []string) error {
	councils := persona.ListCouncils()
	for _, c := range councils {
		fmt.Printf("  %s — %s (%d members, %s)\n", c.Name, c.Description, len(c.Members), c.Strategy)
		for _, m := range c.Members {
			fmt.Printf("    • %s\n", m.Name)
		}
		fmt.Printf("    Chair: %s\n\n", c.Chair.Name)
	}
	return nil
}

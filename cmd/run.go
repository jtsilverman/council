package cmd

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/jtsilverman/council/internal/config"
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

// getCouncil looks up a council by name from built-ins and custom YAML configs.
func getCouncil(name string) (*council.Council, error) {
	// Try built-in first
	c, err := persona.GetCouncil(name)
	if err == nil {
		return c, nil
	}

	// Try custom councils
	customs, _ := config.LoadCustomCouncils(flagConfig)
	for _, cc := range customs {
		if cc.Name == name {
			return cc, nil
		}
	}

	return nil, fmt.Errorf("unknown council %q. Run 'council list' to see available councils", name)
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
	c, err := getCouncil(flagCouncil)
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
		switch flagProvider {
		case "openai":
			p, err = provider.NewOpenAIProvider()
		case "gemini":
			p, err = provider.NewGeminiProvider()
		case "openrouter":
			p, err = provider.NewOpenRouterProvider()
		case "ollama":
			p = provider.NewOllamaProvider("")
		default:
			p, err = provider.NewAnthropicProvider()
		}
		if err != nil {
			return fmt.Errorf("create %s provider: %w", flagProvider, err)
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
	// Built-in councils
	councils := persona.ListCouncils()
	fmt.Println("Built-in:")
	for _, c := range councils {
		fmt.Printf("  %s — %s (%d members, %s)\n", c.Name, c.Description, len(c.Members), c.Strategy)
		for _, m := range c.Members {
			fmt.Printf("    - %s\n", m.Name)
		}
		fmt.Printf("    Chair: %s\n\n", c.Chair.Name)
	}

	// Custom councils
	customs, _ := config.LoadCustomCouncils(flagConfig)
	if len(customs) > 0 {
		fmt.Println("Custom:")
		for _, c := range customs {
			fmt.Printf("  %s — %s (%d members, %s)\n", c.Name, c.Description, len(c.Members), c.Strategy)
			for _, m := range c.Members {
				fmt.Printf("    - %s\n", m.Name)
			}
			fmt.Printf("    Chair: %s\n\n", c.Chair.Name)
		}
	}

	return nil
}

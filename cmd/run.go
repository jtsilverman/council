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
	c, err := persona.GetCouncil(name)
	if err == nil {
		return c, nil
	}
	customs, _ := config.LoadCustomCouncils(flagConfig)
	for _, cc := range customs {
		if cc.Name == name {
			return cc, nil
		}
	}
	return nil, fmt.Errorf("unknown council %q. Run 'council list' to see available councils", name)
}

// buildProviders creates a Providers struct from flags.
func buildProviders() (*council.Providers, error) {
	if flagModels != "" {
		// Multi-model mode: auto-detect provider per model
		models := strings.Split(flagModels, ",")
		providers := &council.Providers{
			Default:  provider.DetectDefault(),
			PerModel: make(map[int]provider.Provider),
		}
		for i, m := range models {
			m = strings.TrimSpace(m)
			p, err := provider.DetectProvider(m)
			if err != nil {
				return nil, fmt.Errorf("model %q: %w", m, err)
			}
			providers.PerModel[i] = p
		}
		return providers, nil
	}

	// Single model mode (default or --model override)
	var p provider.Provider
	if flagModel != "" {
		var err error
		p, err = provider.DetectProvider(flagModel)
		if err != nil {
			return nil, err
		}
	} else {
		p = provider.DetectDefault()
	}
	return &council.Providers{Default: p}, nil
}

// applyMemberFlags adjusts the council members based on --light/--all/--with flags.
func applyMemberFlags(c *council.Council) {
	if c.Name != "code-review" {
		return
	}

	if flagWith != "" {
		names := strings.Split(flagWith, ",")
		for i := range names {
			names[i] = strings.TrimSpace(names[i])
		}
		members := persona.GetMembersByNames(names)
		if len(members) > 0 {
			c.Members = members
		}
	} else if flagAll {
		c.Members = persona.AllMembers()
	} else if flagLight {
		c.Members = persona.LightMembers()
	}

	// Apply model names from --models to members (round-robin)
	if flagModels != "" {
		models := strings.Split(flagModels, ",")
		for i := range c.Members {
			c.Members[i].Model = strings.TrimSpace(models[i%len(models)])
		}
	} else if flagModel != "" {
		for i := range c.Members {
			c.Members[i].Model = flagModel
		}
		c.Chair.Model = flagModel
	}
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

	// Apply strategy flags
	if flagDeep {
		c.Strategy = "debate"
	} else if flagStrategy != "" {
		c.Strategy = flagStrategy
	}

	// Apply member selection flags
	applyMemberFlags(c)

	// Create providers
	providers, err := buildProviders()
	if err != nil {
		return err
	}

	// Get strategy
	strat := strategy.Get(c.Strategy)

	// Run deliberation
	fmt.Fprintf(os.Stderr, "Running %s council (%d members, %s strategy)...\n", c.Name, len(c.Members), c.Strategy)
	delib, err := council.Run(cmd.Context(), c, query, providers, strat)
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
	fmt.Println("Built-in:")
	for _, c := range councils {
		fmt.Printf("  %s — %s (%d members, %s)\n", c.Name, c.Description, len(c.Members), c.Strategy)
		for _, m := range c.Members {
			fmt.Printf("    - %s\n", m.Name)
		}
		fmt.Printf("    Chair: %s\n\n", c.Chair.Name)
	}

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

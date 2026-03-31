package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/jtsilverman/council/internal/council"
	"gopkg.in/yaml.v3"
)

type yamlCouncil struct {
	Name        string       `yaml:"name"`
	Description string       `yaml:"description"`
	Strategy    string       `yaml:"strategy"`
	MaxRounds   int          `yaml:"max_rounds"`
	Members     []yamlMember `yaml:"members"`
	Chair       yamlMember   `yaml:"chair"`
}

type yamlMember struct {
	Name     string `yaml:"name"`
	Persona  string `yaml:"persona"`
	Provider string `yaml:"provider"`
	Model    string `yaml:"model"`
}

// LoadCustomCouncils searches for YAML council definitions and returns them.
// Search order: explicit path, .council.yaml in cwd, ~/.config/council/councils/*.yaml
func LoadCustomCouncils(explicitPath string) ([]*council.Council, error) {
	var paths []string

	if explicitPath != "" {
		paths = append(paths, explicitPath)
	}

	// Check current directory
	if _, err := os.Stat(".council.yaml"); err == nil {
		paths = append(paths, ".council.yaml")
	}

	// Check ~/.config/council/councils/
	home, _ := os.UserHomeDir()
	if home != "" {
		configDir := filepath.Join(home, ".config", "council", "councils")
		entries, err := os.ReadDir(configDir)
		if err == nil {
			for _, e := range entries {
				if !e.IsDir() && (filepath.Ext(e.Name()) == ".yaml" || filepath.Ext(e.Name()) == ".yml") {
					paths = append(paths, filepath.Join(configDir, e.Name()))
				}
			}
		}
	}

	var councils []*council.Council
	seen := map[string]bool{}
	for _, p := range paths {
		c, err := loadYAML(p)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: skipping %s: %v\n", p, err)
			continue
		}
		if !seen[c.Name] {
			councils = append(councils, c)
			seen[c.Name] = true
		}
	}
	return councils, nil
}

func loadYAML(path string) (*council.Council, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var yc yamlCouncil
	if err := yaml.Unmarshal(data, &yc); err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}

	if yc.Name == "" {
		return nil, fmt.Errorf("%s: name is required", path)
	}

	c := &council.Council{
		Name:        yc.Name,
		Description: yc.Description,
		Strategy:    yc.Strategy,
		MaxRounds:   yc.MaxRounds,
	}
	if c.Strategy == "" {
		c.Strategy = "debate"
	}
	if c.MaxRounds == 0 {
		c.MaxRounds = 1
	}

	for _, m := range yc.Members {
		c.Members = append(c.Members, council.Member{
			Name:     m.Name,
			Persona:  m.Persona,
			Provider: m.Provider,
			Model:    m.Model,
		})
	}

	c.Chair = council.Member{
		Name:     yc.Chair.Name,
		Persona:  yc.Chair.Persona,
		Provider: yc.Chair.Provider,
		Model:    yc.Chair.Model,
	}
	if c.Chair.Name == "" {
		c.Chair.Name = "Chair"
		c.Chair.Persona = "Synthesize the council members' responses into a clear, decisive answer."
	}

	return c, nil
}

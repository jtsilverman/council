package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadYAML_ValidFullFields(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.yaml")
	data := `name: test-council
description: A test council
strategy: vote
max_rounds: 3
members:
  - name: Alice
    persona: You are Alice.
    provider: anthropic
    model: claude-sonnet-4-20250514
  - name: Bob
    persona: You are Bob.
    provider: openai
    model: gpt-4
chair:
  name: Chair
  persona: You synthesize.
  provider: anthropic
  model: claude-sonnet-4-20250514
`
	if err := os.WriteFile(path, []byte(data), 0644); err != nil {
		t.Fatal(err)
	}

	c, err := loadYAML(path)
	if err != nil {
		t.Fatalf("loadYAML() error: %v", err)
	}

	if c.Name != "test-council" {
		t.Errorf("Name = %q, want test-council", c.Name)
	}
	if c.Description != "A test council" {
		t.Errorf("Description = %q", c.Description)
	}
	if c.Strategy != "vote" {
		t.Errorf("Strategy = %q, want vote", c.Strategy)
	}
	if c.MaxRounds != 3 {
		t.Errorf("MaxRounds = %d, want 3", c.MaxRounds)
	}
	if len(c.Members) != 2 {
		t.Fatalf("Members = %d, want 2", len(c.Members))
	}
	if c.Members[0].Name != "Alice" {
		t.Errorf("Members[0].Name = %q", c.Members[0].Name)
	}
	if c.Members[0].Provider != "anthropic" {
		t.Errorf("Members[0].Provider = %q", c.Members[0].Provider)
	}
	if c.Chair.Name != "Chair" {
		t.Errorf("Chair.Name = %q", c.Chair.Name)
	}
}

func TestLoadYAML_MissingNameErrors(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "noname.yaml")
	data := `description: No name here
members:
  - name: Alice
    persona: You are Alice.
`
	if err := os.WriteFile(path, []byte(data), 0644); err != nil {
		t.Fatal(err)
	}

	_, err := loadYAML(path)
	if err == nil {
		t.Fatal("expected error for missing name, got nil")
	}
}

func TestLoadYAML_StrategyDefaults(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "defaults.yaml")
	data := `name: minimal
members:
  - name: Alice
    persona: You are Alice.
`
	if err := os.WriteFile(path, []byte(data), 0644); err != nil {
		t.Fatal(err)
	}

	c, err := loadYAML(path)
	if err != nil {
		t.Fatalf("loadYAML() error: %v", err)
	}

	if c.Strategy != "debate" {
		t.Errorf("Strategy = %q, want debate (default)", c.Strategy)
	}
}

func TestLoadYAML_MaxRoundsDefaults(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "defaults.yaml")
	data := `name: minimal
members:
  - name: Alice
    persona: You are Alice.
`
	if err := os.WriteFile(path, []byte(data), 0644); err != nil {
		t.Fatal(err)
	}

	c, err := loadYAML(path)
	if err != nil {
		t.Fatalf("loadYAML() error: %v", err)
	}

	if c.MaxRounds != 1 {
		t.Errorf("MaxRounds = %d, want 1 (default)", c.MaxRounds)
	}
}

func TestLoadYAML_ChairDefaults(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "nochair.yaml")
	data := `name: minimal
members:
  - name: Alice
    persona: You are Alice.
`
	if err := os.WriteFile(path, []byte(data), 0644); err != nil {
		t.Fatal(err)
	}

	c, err := loadYAML(path)
	if err != nil {
		t.Fatalf("loadYAML() error: %v", err)
	}

	if c.Chair.Name != "Chair" {
		t.Errorf("Chair.Name = %q, want Chair (default)", c.Chair.Name)
	}
	if c.Chair.Persona == "" {
		t.Error("Chair.Persona should have default value")
	}
}

func TestLoadYAML_InvalidYAML(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "bad.yaml")
	data := `name: [this is not valid yaml
  broken: {{{
`
	if err := os.WriteFile(path, []byte(data), 0644); err != nil {
		t.Fatal(err)
	}

	_, err := loadYAML(path)
	if err == nil {
		t.Fatal("expected error for invalid YAML, got nil")
	}
}

func TestLoadYAML_EmptyFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "empty.yaml")
	if err := os.WriteFile(path, []byte(""), 0644); err != nil {
		t.Fatal(err)
	}

	_, err := loadYAML(path)
	if err == nil {
		t.Fatal("expected error for empty file (no name), got nil")
	}
}

func TestLoadCustomCouncils_ExplicitPath(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "custom.yaml")
	data := `name: custom-council
description: Custom
members:
  - name: Tester
    persona: You test things.
`
	if err := os.WriteFile(path, []byte(data), 0644); err != nil {
		t.Fatal(err)
	}

	councils, err := LoadCustomCouncils(path)
	if err != nil {
		t.Fatalf("LoadCustomCouncils() error: %v", err)
	}

	if len(councils) == 0 {
		t.Fatal("expected at least 1 council")
	}

	found := false
	for _, c := range councils {
		if c.Name == "custom-council" {
			found = true
		}
	}
	if !found {
		t.Error("custom-council not found in results")
	}
}

func TestLoadCustomCouncils_NoExplicitPath(t *testing.T) {
	// With no explicit path and no .council.yaml in cwd, should return empty (no error)
	councils, err := LoadCustomCouncils("")
	if err != nil {
		t.Fatalf("LoadCustomCouncils() error: %v", err)
	}
	// Result depends on environment but should not error
	_ = councils
}

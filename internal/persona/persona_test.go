package persona

import "testing"

func TestGetCouncil_ReturnsCopy(t *testing.T) {
	c1, err := GetCouncil("general")
	if err != nil {
		t.Fatalf("GetCouncil(general) error: %v", err)
	}

	// Mutate the returned council
	originalName := c1.Members[0].Name
	c1.Members[0].Name = "MUTATED"
	c1.Members = append(c1.Members, c1.Members[0])

	// Get again and verify it's unchanged
	c2, err := GetCouncil("general")
	if err != nil {
		t.Fatalf("GetCouncil(general) second call error: %v", err)
	}

	if c2.Members[0].Name != originalName {
		t.Errorf("Members[0].Name = %q, want %q (mutation leaked)", c2.Members[0].Name, originalName)
	}
}

func TestGetCouncil_UnknownReturnsError(t *testing.T) {
	_, err := GetCouncil("nonexistent-council")
	if err == nil {
		t.Fatal("expected error for unknown council, got nil")
	}
}

func TestListCouncils_NonEmpty(t *testing.T) {
	councils := ListCouncils()
	if len(councils) == 0 {
		t.Fatal("ListCouncils() returned empty list")
	}
}

func TestListCouncils_RequiredFields(t *testing.T) {
	councils := ListCouncils()
	for _, c := range councils {
		if c.Name == "" {
			t.Error("council has empty Name")
		}
		if c.Description == "" {
			t.Errorf("council %q has empty Description", c.Name)
		}
		if len(c.Members) == 0 {
			t.Errorf("council %q has no Members", c.Name)
		}
		if c.Strategy == "" {
			t.Errorf("council %q has empty Strategy", c.Name)
		}
	}
}

func TestListCouncils_ContainsKnownCouncils(t *testing.T) {
	councils := ListCouncils()
	names := map[string]bool{}
	for _, c := range councils {
		names[c.Name] = true
	}

	for _, want := range []string{"code-review", "general", "writing"} {
		if !names[want] {
			t.Errorf("ListCouncils() missing %q", want)
		}
	}
}

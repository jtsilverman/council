package provider

// PromptProfile defines per-provider budget constraints for prompt sizing.
type PromptProfile struct {
	ReviewBudgetChars         int
	DebateBudgetChars         int
	SynthesisBudgetChars      int
	RequireStructuredDigest   bool
	PreferWorkspaceRead       bool
}

// DefaultProfile returns generous defaults suitable for most providers.
func DefaultProfile() PromptProfile {
	return PromptProfile{
		ReviewBudgetChars:       100_000,
		DebateBudgetChars:       100_000,
		SynthesisBudgetChars:    100_000,
		RequireStructuredDigest: false,
		PreferWorkspaceRead:     false,
	}
}

// CodexProfile returns tight budgets for Codex's limited context window.
func CodexProfile() PromptProfile {
	return PromptProfile{
		ReviewBudgetChars:       8_000,
		DebateBudgetChars:       6_000,
		SynthesisBudgetChars:    6_000,
		RequireStructuredDigest: true,
		PreferWorkspaceRead:     true,
	}
}

// CodexRetryProfile returns halved budgets for Codex retry attempts.
func CodexRetryProfile() PromptProfile {
	return PromptProfile{
		ReviewBudgetChars:       4_000,
		DebateBudgetChars:       3_000,
		SynthesisBudgetChars:    3_000,
		RequireStructuredDigest: true,
		PreferWorkspaceRead:     true,
	}
}

// ProfileFor returns the appropriate PromptProfile for the given provider.
func ProfileFor(p Provider) PromptProfile {
	if p.Name() == "codex" {
		return CodexProfile()
	}
	return DefaultProfile()
}

package council

// Member represents one council member with a specific expert persona.
type Member struct {
	Name     string // "Security Auditor", "Performance Engineer"
	Persona  string // System prompt defining their expertise
	Provider string // "anthropic", "cli", etc.
	Model    string // "claude-sonnet-4-20250514", etc.
}

// Council is a named group of members with a deliberation strategy.
type Council struct {
	Name        string
	Description string
	Members     []Member
	Chair       Member
	Strategy    string // "debate", "vote", "ranked"
	MaxRounds   int    // For debate: max debate rounds (default 1)
}

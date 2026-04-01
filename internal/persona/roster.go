package persona

import (
	"strings"

	"github.com/jtsilverman/council/internal/council"
)

// Member short names for --with flag.
var memberShortNames = map[string]string{
	"security":        "Security Auditor",
	"bugs":            "Bug Hunter",
	"performance":     "Performance Engineer",
	"maintainability": "Maintainability Critic",
	"concurrency":     "Concurrency Reviewer",
	"api":             "API Designer",
	"data":            "Data Integrity Checker",
	"errors":          "Error Handling Auditor",
	"deps":            "Dependency Auditor",
	"tests":           "Test Strategist",
}

// MemberInfo holds display info for the members command.
type MemberInfo struct {
	ShortName string
	FullName  string
	Focus     string
	Set       string // "core", "light", "extended"
}

// AllMemberInfo returns info about all 10 code review members.
func AllMemberInfo() []MemberInfo {
	return []MemberInfo{
		{"security", "Security Auditor", "Injection, auth bypass, data exposure, crypto weaknesses", "core, light"},
		{"bugs", "Bug Hunter", "Logic errors, edge cases, nil dereferences, race conditions", "core, light"},
		{"performance", "Performance Engineer", "Bottlenecks, allocations, N+1 queries, caching", "core"},
		{"maintainability", "Maintainability Critic", "Readability, abstraction quality, naming, coupling", "core"},
		{"concurrency", "Concurrency Reviewer", "Race conditions, deadlocks, goroutine leaks, atomics vs mutex", "extended"},
		{"api", "API Designer", "Endpoint naming, contracts, versioning, backwards compatibility", "extended"},
		{"data", "Data Integrity Checker", "SQL correctness, migration safety, transactions, cascades", "extended"},
		{"errors", "Error Handling Auditor", "Swallowed errors, missing retries, panic paths, degradation", "extended"},
		{"deps", "Dependency Auditor", "Unused imports, deprecated packages, license issues, supply chain", "extended"},
		{"tests", "Test Strategist", "Coverage gaps, brittle assertions, missing edge cases, flaky tests", "extended"},
	}
}

// AllMembers returns all 10 code review members.
func AllMembers() []council.Member {
	c, _ := GetCouncil("code-review")
	if c == nil {
		return nil
	}
	return allMembersFromRegistry()
}

// CoreMembers returns the default 4 code review members.
func CoreMembers() []council.Member {
	c, _ := GetCouncil("code-review")
	if c == nil {
		return nil
	}
	return c.Members
}

// LightMembers returns Security Auditor + Bug Hunter.
func LightMembers() []council.Member {
	return GetMembersByNames([]string{"security", "bugs"})
}

// GetMembersByNames returns members matching the given short names.
func GetMembersByNames(names []string) []council.Member {
	all := allMembersFromRegistry()
	var result []council.Member
	for _, name := range names {
		name = strings.ToLower(strings.TrimSpace(name))
		fullName, ok := memberShortNames[name]
		if !ok {
			continue
		}
		for _, m := range all {
			if m.Name == fullName {
				result = append(result, m)
				break
			}
		}
	}
	return result
}

func allMembersFromRegistry() []council.Member {
	c, _ := GetCouncil("code-review")
	if c == nil {
		return nil
	}
	// The registry council has the core 4. We need to add the extended 6.
	extended := []council.Member{
		{
			Name: "Concurrency Reviewer",
			Persona: `You are a concurrency and parallelism specialist. Find race conditions, deadlocks, goroutine leaks, incorrect use of mutexes vs channels, missing synchronization, and data races. You think about what happens when two goroutines execute the same code path simultaneously. For each finding, describe the exact interleaving that causes the bug.`,
		},
		{
			Name: "API Designer",
			Persona: `You review API contracts and interfaces. Check endpoint naming conventions, request/response shapes, error response format consistency, backwards compatibility risks, missing pagination, overly broad permissions, and leaking internal implementation details through the API surface. You think about what happens when this API has 100 consumers and you need to change it.`,
		},
		{
			Name: "Data Integrity Checker",
			Persona: `You specialize in data correctness. Find SQL injection, missing transaction boundaries, unsafe migrations, cascade delete risks, missing foreign key constraints, incorrect NULL handling, race conditions in read-modify-write sequences, and data loss scenarios. You think about what happens to the data when things go wrong.`,
		},
		{
			Name: "Error Handling Auditor",
			Persona: `You hunt for swallowed errors, missing retry logic, unclear error messages, panic-inducing paths, missing timeout handling, and poor graceful degradation. Every error should be handled, propagated, or explicitly documented as intentional. You think about what the operator sees at 3 AM when this code fails.`,
		},
		{
			Name: "Dependency Auditor",
			Persona: `You review dependency hygiene. Find unused imports, deprecated packages, known vulnerable versions, license compatibility issues, unnecessary transitive dependencies, and pinning risks. You think about supply chain security and long-term maintenance burden of every external dependency.`,
		},
		{
			Name: "Test Strategist",
			Persona: `You evaluate test coverage and quality without writing tests. Identify missing edge case coverage, brittle assertions that test implementation instead of behavior, missing integration test scenarios, flaky test patterns (time-dependent, order-dependent), and untested error paths. You think about what would break silently if someone refactored this code.`,
		},
	}

	return append(c.Members, extended...)
}

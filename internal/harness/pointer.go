package harness

import (
	"fmt"
	"os"
	"path/filepath"
)

// KnownPointerTargets maps harness names to their default guidance file.
var KnownPointerTargets = map[string]string{
	"claude": "CLAUDE.md",
	"codex":  "AGENTS.md",
	"jcode":  "AGENTS.md",
}

// SyncPointers creates or updates pointer files that redirect to the contract.
// Each target becomes a file containing "@AGENTS.md" (or whatever the contract is).
// If the target already exists and is not a union pointer, it is left untouched.
func SyncPointers(shopDir, contract string, targets []string) ([]string, error) {
	var created []string
	for _, target := range targets {
		if target == contract {
			continue
		}
		targetPath := filepath.Join(shopDir, target)

		if existing, err := os.ReadFile(targetPath); err == nil {
			if !isUnionPointer(existing) {
				return created, fmt.Errorf("%s already exists and is not a union pointer — remove it first or skip", target)
			}
		}

		content := fmt.Sprintf("@%s\n", contract)
		if err := os.WriteFile(targetPath, []byte(content), 0o644); err != nil {
			return created, fmt.Errorf("write pointer %s: %w", target, err)
		}
		created = append(created, target)
	}
	return created, nil
}

// isUnionPointer checks if file content is a union pointer (starts with @).
func isUnionPointer(content []byte) bool {
	return len(content) > 0 && content[0] == '@'
}

// DefaultPointerTargets returns sensible pointer targets for detected harnesses
// that use a different guidance file than the contract.
func DefaultPointerTargets(contract string, adapters []Adapter) []string {
	seen := map[string]bool{}
	var targets []string
	for _, a := range adapters {
		t, ok := KnownPointerTargets[a.Name()]
		if !ok || t == contract || seen[t] {
			continue
		}
		seen[t] = true
		targets = append(targets, t)
	}
	return targets
}

package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/chazu/union/internal/harness"
	"github.com/chazu/union/internal/paths"
	"github.com/chazu/union/internal/qpath"
	"github.com/chazu/union/internal/store"
	"github.com/spf13/cobra"
)

func newEmitCmd() *cobra.Command {
	var (
		write       bool
		harnessName string
	)
	cmd := &cobra.Command{
		Use:   "emit",
		Short: "Render hook clauses into native harness config files.",
		Long: `Emit reads hook clauses ratified in the current shop and translates them
into native configuration for each detected or configured harness.

By default, shows a preview of changes (dry-run). Use --write to apply.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runEmit(cmd, write, harnessName)
		},
	}
	cmd.Flags().BoolVar(&write, "write", false, "Apply changes (default is dry-run)")
	cmd.Flags().StringVar(&harnessName, "harness", "", "Emit only to this harness")
	return cmd
}

func runEmit(cmd *cobra.Command, write bool, filterHarness string) error {
	s, contractPath, err := currentShop()
	if err != nil {
		return err
	}
	w := cmd.OutOrStdout()

	// Resolve harnesses.
	adapters, err := harness.ResolveHarnesses(s.Dir)
	if err != nil {
		return err
	}
	if filterHarness != "" {
		var filtered []harness.Adapter
		for _, a := range adapters {
			if a.Name() == filterHarness {
				filtered = append(filtered, a)
			}
		}
		if len(filtered) == 0 {
			return fmt.Errorf("harness %q not found for this shop", filterHarness)
		}
		adapters = filtered
	}
	if len(adapters) == 0 {
		fmt.Fprintln(w, "No harnesses detected. Run 'union harness add <name>' or create a union.toml.")
		return nil
	}

	// Collect hook clauses from the contract.
	hooks, warnings, err := collectHookClauses(contractPath)
	if err != nil {
		return err
	}
	for _, warn := range warnings {
		fmt.Fprintf(cmd.ErrOrStderr(), "warning: %s\n", warn)
	}
	if len(hooks) == 0 {
		fmt.Fprintln(w, "No hook clauses ratified in this shop.")
		return nil
	}

	unionDir, err := paths.UnionDir()
	if err != nil {
		return err
	}

	fmt.Fprintf(w, "Shop: %s\n", s.Dir)
	fmt.Fprintf(w, "Harnesses: %s\n\n", joinAdapterNames(adapters))

	// Emit per harness.
	for _, adapter := range adapters {
		fmt.Fprintf(w, "── %s (%s) ──\n", adapter.Name(), adapter.SettingsPath())

		// Resolve template variables for this harness.
		vars := harness.ResolveVars(s.Dir, unionDir, adapter.Name())

		// Filter hooks for this harness and expand templates.
		var applicable []harness.Hook
		for _, h := range hooks {
			if !adapter.Supports(h.Event) {
				if h.Degrade == "error" {
					return fmt.Errorf("%s does not support event %q (degrade=error)", adapter.Name(), h.Event)
				}
				if h.Degrade == "warn" {
					fmt.Fprintf(w, "  ⚠ %s: not supported (degrade: warn)\n", h.Event)
				}
				continue
			}
			applicable = append(applicable, vars.ExpandHook(h))
		}

		if len(applicable) == 0 {
			fmt.Fprintln(w, "  (no applicable hooks)")
			continue
		}

		// Read existing config.
		settingsPath := filepath.Join(s.Dir, adapter.SettingsPath())
		existing, _ := os.ReadFile(settingsPath)

		// Emit.
		output, err := adapter.Emit(applicable, existing)
		if err != nil {
			return fmt.Errorf("emit %s: %w", adapter.Name(), err)
		}

		for _, h := range applicable {
			fmt.Fprintf(w, "  + %s: %s\n", h.Event, truncate(h.Command, 60))
		}

		if write {
			dir := filepath.Dir(settingsPath)
			if err := os.MkdirAll(dir, 0o755); err != nil {
				return fmt.Errorf("mkdir %s: %w", dir, err)
			}
			if err := os.WriteFile(settingsPath, output, 0o644); err != nil {
				return fmt.Errorf("write %s: %w", settingsPath, err)
			}
			fmt.Fprintf(w, "  ✓ wrote %s\n", adapter.SettingsPath())
		}
		fmt.Fprintln(w)
	}

	if !write {
		fmt.Fprintln(w, "Dry run. Use --write to apply changes.")
	}
	return nil
}

// collectHookClauses reads the contract, finds all ratified clauses that
// are hook-type (start with frontmatter type: hook), and returns them as
// normalized Hooks.
func collectHookClauses(contractPath string) ([]harness.Hook, []string, error) {
	contract, err := os.ReadFile(contractPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil, nil
		}
		return nil, nil, fmt.Errorf("read contract: %w", err)
	}

	blocks, err := parseContractBlocks(contract)
	if err != nil {
		return nil, nil, fmt.Errorf("parse contract: %w", err)
	}

	var hooks []harness.Hook
	var warnings []string
	for _, b := range blocks {
		// Try parsing as a hook clause. Non-hook clauses are silently skipped.
		fm, command, err := harness.ParseHookClause(b.body)
		if err != nil {
			continue
		}

		// If clause specifies harness filters, note it.
		hook := fm.ToHook(command)

		// Also try reading from store to get freshest version.
		q, qerr := qpath.Parse(b.path)
		if qerr == nil {
			freshBody, ferr := readFromStore(q)
			if ferr == nil {
				if freshFm, freshCmd, perr := harness.ParseHookClause(freshBody); perr == nil {
					hook = freshFm.ToHook(freshCmd)
				}
			}
		}

		hooks = append(hooks, hook)
	}
	return hooks, warnings, nil
}

// readFromStore reads the current clause body from the store.
func readFromStore(q qpath.Qualified) ([]byte, error) {
	unionDir, err := paths.UnionDir()
	if err != nil {
		return nil, err
	}
	s, err := store.OpenNamed(unionDir, q.Store)
	if err != nil {
		return nil, err
	}
	return s.Get(q.Path)
}

// parseContractBlocks is a minimal contract parser that extracts marked blocks.
type contractBlock struct {
	path string
	body []byte
}

func parseContractBlocks(contract []byte) ([]contractBlock, error) {
	// Reuse the shop package's parser via the block structure.
	// We do a simple inline parse to avoid circular deps.
	lines := strings.Split(string(contract), "\n")
	var blocks []contractBlock
	var current *contractBlock
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "<!-- BEGIN union:") && strings.HasSuffix(trimmed, "-->") {
			path := strings.TrimPrefix(trimmed, "<!-- BEGIN union:")
			path = strings.TrimSuffix(path, "-->")
			path = strings.TrimSpace(path)
			current = &contractBlock{path: path}
		} else if strings.HasPrefix(trimmed, "<!-- END union:") && current != nil {
			blocks = append(blocks, *current)
			current = nil
		} else if current != nil {
			current.body = append(current.body, []byte(line+"\n")...)
		}
	}
	return blocks, nil
}

func joinAdapterNames(adapters []harness.Adapter) string {
	names := make([]string, len(adapters))
	for i, a := range adapters {
		names[i] = a.Name()
	}
	return strings.Join(names, ", ")
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n-3] + "..."
}

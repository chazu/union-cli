package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/chazu/union/internal/harness"
	"github.com/spf13/cobra"
)

func newPointerCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "pointer",
		Short: "Manage guidance file pointers (e.g., CLAUDE.md → AGENTS.md).",
		Long: `Create pointer files that redirect harness-specific guidance files
to the canonical contract. For example, make CLAUDE.md point to AGENTS.md
so Claude Code reads the same contract as other harnesses.

A pointer file contains "@AGENTS.md" — a convention that tells agents
to read the referenced file instead.`,
	}
	cmd.AddCommand(
		newPointerSyncCmd(),
		newPointerListCmd(),
	)
	return cmd
}

func newPointerSyncCmd() *cobra.Command {
	var targets []string
	cmd := &cobra.Command{
		Use:   "sync",
		Short: "Create or update pointer files for detected harnesses.",
		Long: `Creates pointer files based on union.toml [pointers] config or
auto-detected harness defaults. For example, if the contract is AGENTS.md
and Claude Code is detected, creates CLAUDE.md containing "@AGENTS.md".`,
		RunE: func(cmd *cobra.Command, args []string) error {
			s, _, err := currentShop()
			if err != nil {
				return err
			}
			w := cmd.OutOrStdout()

			cfg, err := harness.LoadConfig(s.Dir)
			if err != nil {
				return err
			}

			// Use explicit targets from config or flags, else auto-detect.
			effectiveTargets := targets
			if len(effectiveTargets) == 0 {
				effectiveTargets = cfg.Pointers.Targets
			}
			if len(effectiveTargets) == 0 {
				adapters, err := harness.ResolveHarnesses(s.Dir)
				if err != nil {
					return err
				}
				effectiveTargets = harness.DefaultPointerTargets(s.Contract, adapters)
			}

			if len(effectiveTargets) == 0 {
				fmt.Fprintln(w, "No pointer targets needed (contract matches all detected harnesses).")
				return nil
			}

			created, err := harness.SyncPointers(s.Dir, s.Contract, effectiveTargets)
			if err != nil {
				return err
			}
			for _, t := range created {
				fmt.Fprintf(w, "  → %s points to %s\n", t, s.Contract)
			}
			if len(created) == 0 {
				fmt.Fprintln(w, "All pointers up to date.")
			}
			return nil
		},
	}
	cmd.Flags().StringSliceVar(&targets, "target", nil, "Specific files to create as pointers")
	return cmd
}

func newPointerListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "Show existing pointer files in the current shop.",
		RunE: func(cmd *cobra.Command, args []string) error {
			s, _, err := currentShop()
			if err != nil {
				return err
			}
			w := cmd.OutOrStdout()

			// Scan for common guidance filenames that might be pointers.
			candidates := []string{"CLAUDE.md", "AGENTS.md", "CODEX.md", "JCODE.md", ".cursorrules"}
			found := false
			for _, name := range candidates {
				p := filepath.Join(s.Dir, name)
				data, err := os.ReadFile(p)
				if err != nil {
					continue
				}
				content := strings.TrimSpace(string(data))
				if len(content) > 0 && content[0] == '@' {
					target := strings.TrimPrefix(content, "@")
					fmt.Fprintf(w, "  %s → %s\n", name, target)
					found = true
				}
			}
			if !found {
				fmt.Fprintln(w, "No pointer files found.")
			}
			return nil
		},
	}
}

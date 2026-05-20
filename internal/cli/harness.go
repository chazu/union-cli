package cli

import (
	"fmt"

	"github.com/chazu/union/internal/harness"
	"github.com/spf13/cobra"
)

func newHarnessCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "harness",
		Short: "Manage AI coding harness integrations.",
		Long:  "Detect, list, and configure which AI coding harnesses (Claude Code, OpenCode, etc.) are active in the current shop.",
	}
	cmd.AddCommand(
		newHarnessDetectCmd(),
		newHarnessListCmd(),
		newHarnessAddCmd(),
		newHarnessRemoveCmd(),
	)
	return cmd
}

func newHarnessDetectCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "detect",
		Short: "Auto-detect harnesses in the current shop.",
		RunE: func(cmd *cobra.Command, args []string) error {
			s, _, err := currentShop()
			if err != nil {
				return err
			}
			found := harness.Detect(s.Dir)
			if len(found) == 0 {
				fmt.Fprintln(cmd.OutOrStdout(), "No harnesses detected.")
				return nil
			}
			for _, a := range found {
				fmt.Fprintf(cmd.OutOrStdout(), "  %s  (%s)\n", a.Name(), a.SettingsPath())
			}
			return nil
		},
	}
}

func newHarnessListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List configured or detected harnesses.",
		RunE: func(cmd *cobra.Command, args []string) error {
			s, _, err := currentShop()
			if err != nil {
				return err
			}
			adapters, err := harness.ResolveHarnesses(s.Dir)
			if err != nil {
				return err
			}
			if len(adapters) == 0 {
				fmt.Fprintln(cmd.OutOrStdout(), "No harnesses configured or detected.")
				return nil
			}
			w := cmd.OutOrStdout()
			for _, a := range adapters {
				caps := a.Capabilities()
				fmt.Fprintf(w, "%-12s  settings: %-30s  events: %v\n", a.Name(), a.SettingsPath(), caps)
			}
			return nil
		},
	}
}

func newHarnessAddCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "add <name>",
		Short: "Explicitly declare a harness for this shop.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]
			a := harness.ByName(name)
			if a == nil {
				return fmt.Errorf("unknown harness: %q (known: claude, opencode, codex, jcode)", name)
			}

			s, _, err := currentShop()
			if err != nil {
				return err
			}
			cfg, err := harness.LoadConfig(s.Dir)
			if err != nil {
				return err
			}
			if _, exists := cfg.Harnesses[name]; exists {
				return fmt.Errorf("harness %q already configured", name)
			}
			cfg.Harnesses[name] = harness.HarnessConfig{
				Settings: a.SettingsPath(),
			}
			if err := harness.SaveConfig(s.Dir, cfg); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Added harness: %s\n", name)
			return nil
		},
	}
}

func newHarnessRemoveCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "remove <name>",
		Short: "Remove a harness declaration from this shop.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]
			s, _, err := currentShop()
			if err != nil {
				return err
			}
			cfg, err := harness.LoadConfig(s.Dir)
			if err != nil {
				return err
			}
			if _, exists := cfg.Harnesses[name]; !exists {
				return fmt.Errorf("harness %q not configured", name)
			}
			delete(cfg.Harnesses, name)
			if err := harness.SaveConfig(s.Dir, cfg); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Removed harness: %s\n", name)
			return nil
		},
	}
}

package cli

import (
	"fmt"
	"os"

	"github.com/chazu/union/internal/paths"
	"github.com/chazu/union/internal/qpath"
	"github.com/chazu/union/internal/store"
	"github.com/spf13/cobra"
)

func newInitCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "init [name]",
		Short: "Initialize the union root and a first store (default: 'default').",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := "default"
			if len(args) == 1 {
				name = args[0]
			}
			if err := qpath.ValidateStoreName(name); err != nil {
				return err
			}
			unionDir, err := paths.UnionDir()
			if err != nil {
				return err
			}
			if err := os.MkdirAll(unionDir, 0o755); err != nil {
				return fmt.Errorf("create %s: %w", unionDir, err)
			}
			shopsPath, err := paths.ShopsFile()
			if err != nil {
				return err
			}
			if _, err := os.Stat(shopsPath); os.IsNotExist(err) {
				if err := os.WriteFile(shopsPath, []byte("# union shops registry\n"), 0o644); err != nil {
					return fmt.Errorf("seed shops.toml: %w", err)
				}
			}
			s, err := store.InitNamed(unionDir, name)
			if err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "initialized store %q at %s\n", name, s.Root())
			return nil
		},
	}
}

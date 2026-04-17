package cli

import (
	"fmt"
	"os"

	"github.com/chazu/union/internal/paths"
	"github.com/chazu/union/internal/store"
	"github.com/spf13/cobra"
)

func newInitCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "init",
		Short: "Initialize the union store.",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			dir, err := paths.UnionDir()
			if err != nil {
				return err
			}
			if err := os.MkdirAll(dir, 0o755); err != nil {
				return fmt.Errorf("create %s: %w", dir, err)
			}
			if _, err := store.Init(dir); err != nil {
				return err
			}
			shopsPath, err := paths.ShopsFile()
			if err != nil {
				return err
			}
			if err := os.WriteFile(shopsPath, []byte("# union shops registry\n"), 0o644); err != nil {
				return fmt.Errorf("seed shops.toml: %w", err)
			}
			fmt.Fprintf(cmd.OutOrStdout(), "initialized union store at %s\n", dir)
			return nil
		},
	}
}

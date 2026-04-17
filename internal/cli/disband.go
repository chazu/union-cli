package cli

import (
	"fmt"
	"path/filepath"

	"github.com/chazu/union/internal/paths"
	"github.com/chazu/union/internal/shop"
	"github.com/spf13/cobra"
)

func newDisbandCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "disband <dir>",
		Short: "Unregister a shop.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			abs, err := filepath.Abs(args[0])
			if err != nil {
				return err
			}
			shopsPath, err := paths.ShopsFile()
			if err != nil {
				return err
			}
			r, err := shop.LoadRegistry(shopsPath)
			if err != nil {
				return err
			}
			if err := r.Remove(abs); err != nil {
				return err
			}
			if err := r.Save(); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "disbanded shop %s\n", abs)
			return nil
		},
	}
}

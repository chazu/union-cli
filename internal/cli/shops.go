package cli

import (
	"fmt"

	"github.com/chazu/union/internal/paths"
	"github.com/chazu/union/internal/shop"
	"github.com/spf13/cobra"
)

func newShopsCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "shops",
		Short: "List registered shops.",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			shopsPath, err := paths.ShopsFile()
			if err != nil {
				return err
			}
			r, err := shop.LoadRegistry(shopsPath)
			if err != nil {
				return err
			}
			for _, s := range r.List() {
				fmt.Fprintf(cmd.OutOrStdout(), "%s\t%s\n", s.Dir, s.Contract)
			}
			return nil
		},
	}
}

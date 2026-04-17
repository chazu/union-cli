package cli

import (
	"fmt"
	"os"

	"github.com/chazu/union/internal/shop"
	"github.com/spf13/cobra"
)

func newStrikeCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "strike <path>",
		Short: "Remove a clause from this shop's contract.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			path := args[0]
			_, contractPath, err := currentShop()
			if err != nil {
				return err
			}
			cur, err := os.ReadFile(contractPath)
			if err != nil {
				return err
			}
			out, err := shop.RemoveClause(cur, path)
			if err != nil {
				return err
			}
			if err := os.WriteFile(contractPath, out, 0o644); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "struck %s from %s\n", path, contractPath)
			return nil
		},
	}
}

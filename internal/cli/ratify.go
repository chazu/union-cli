package cli

import (
	"fmt"
	"os"

	"github.com/chazu/union/internal/shop"
	"github.com/spf13/cobra"
)

func newRatifyCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "ratify <path>",
		Short: "Add a clause to this shop's contract.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			path := args[0]
			s, err := openStore()
			if err != nil {
				return err
			}
			if !s.Has(path) {
				return fmt.Errorf("no such clause: %s. See 'union clauses' for available paths.", path)
			}
			body, err := s.Get(path)
			if err != nil {
				return err
			}
			_, contractPath, err := currentShop()
			if err != nil {
				return err
			}
			cur, err := os.ReadFile(contractPath)
			if err != nil && !os.IsNotExist(err) {
				return err
			}
			out, err := shop.InsertClause(cur, path, body)
			if err != nil {
				return err
			}
			if err := os.WriteFile(contractPath, out, 0o644); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "ratified %s into %s\n", path, contractPath)
			return nil
		},
	}
}

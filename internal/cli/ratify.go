package cli

import (
	"fmt"
	"os"

	"github.com/chazu/union/internal/qpath"
	"github.com/chazu/union/internal/shop"
	"github.com/spf13/cobra"
)

func newRatifyCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "ratify <path>",
		Short: "Add a clause to this shop's contract.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			q, err := qpath.Parse(args[0])
			if err != nil {
				return err
			}
			s, err := openStoreFor(q)
			if err != nil {
				return err
			}
			if !s.Has(q.Path) {
				return fmt.Errorf("no such clause: %s. See 'union clauses' for available paths.", q)
			}
			body, err := s.Get(q.Path)
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
			out, err := shop.InsertClause(cur, q.String(), body)
			if err != nil {
				return err
			}
			if err := os.WriteFile(contractPath, out, 0o644); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "ratified %s into %s\n", q, contractPath)
			return nil
		},
	}
}

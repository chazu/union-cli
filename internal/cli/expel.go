package cli

import (
	"fmt"

	"github.com/chazu/union/internal/qpath"
	"github.com/spf13/cobra"
)

func newExpelCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "expel <path>",
		Short: "Remove a clause from the store and strike it from ratified shops.",
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
			if err := s.Delete(q.Path, "expel "+q.String()); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "expelled clause %s\n", q)
			return propagateRemoval(cmd.OutOrStdout(), q.String())
		},
	}
}

package cli

import (
	"fmt"

	"github.com/chazu/union/internal/qpath"
	"github.com/spf13/cobra"
)

func newEditCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "edit <path>",
		Short: "Open a clause in $EDITOR; auto-commits on save and propagates to ratified shops.",
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
			cur, err := s.Get(q.Path)
			if err != nil {
				return err
			}
			body, err := openEditor(cur, q.Path)
			if err != nil {
				return err
			}
			if err := s.Put(q.Path, body, "edit "+q.String()); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "edited clause %s\n", q)
			return propagateUpdate(cmd.OutOrStdout(), q.String(), body)
		},
	}
}

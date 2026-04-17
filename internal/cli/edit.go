package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

func newEditCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "edit <path>",
		Short: "Open a clause in $EDITOR; auto-commits on save and propagates to ratified shops.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			s, err := openStore()
			if err != nil {
				return err
			}
			path := args[0]
			cur, err := s.Get(path)
			if err != nil {
				return err
			}
			body, err := openEditor(cur, path)
			if err != nil {
				return err
			}
			if err := s.Put(path, body, "edit "+path); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "edited clause %s\n", path)
			if err := propagateUpdate(cmd.OutOrStdout(), path, body); err != nil {
				return err
			}
			return nil
		},
	}
}

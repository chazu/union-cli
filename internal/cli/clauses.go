package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

func newClausesCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "clauses [prefix]",
		Short: "List clauses in the store, optionally filtered by path prefix.",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			s, err := openStore()
			if err != nil {
				return err
			}
			prefix := ""
			if len(args) == 1 {
				prefix = args[0]
			}
			all, err := s.List(prefix)
			if err != nil {
				return err
			}
			for _, p := range all {
				fmt.Fprintln(cmd.OutOrStdout(), p)
			}
			return nil
		},
	}
}

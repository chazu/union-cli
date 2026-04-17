package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

func newExpelCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "expel <path>",
		Short: "Remove a clause from the store.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			s, err := openStore()
			if err != nil {
				return err
			}
			path := args[0]
			if err := s.Delete(path, "expel "+path); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "expelled clause %s\n", path)
			return nil
		},
	}
}

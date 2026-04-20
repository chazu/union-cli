package cli

import (
	"github.com/chazu/union/internal/qpath"
	"github.com/spf13/cobra"
)

func newShowCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "show <path>",
		Short: "Print a clause's content.",
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
			body, err := s.Get(q.Path)
			if err != nil {
				return err
			}
			_, err = cmd.OutOrStdout().Write(body)
			return err
		},
	}
}

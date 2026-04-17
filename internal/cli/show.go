package cli

import "github.com/spf13/cobra"

func newShowCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "show <path>",
		Short: "Print a clause's content.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			s, err := openStore()
			if err != nil {
				return err
			}
			body, err := s.Get(args[0])
			if err != nil {
				return err
			}
			_, err = cmd.OutOrStdout().Write(body)
			return err
		},
	}
}

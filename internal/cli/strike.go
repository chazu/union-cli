package cli

import (
	"errors"

	"github.com/spf13/cobra"
)

func newStrikeCmd() *cobra.Command {
	return &cobra.Command{
		Use:    "strike <path>",
		Short:  "Remove a clause from this shop's contract.",
		Hidden: true,
		RunE:   func(*cobra.Command, []string) error { return errors.New("not implemented") },
	}
}

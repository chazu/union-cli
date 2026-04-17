package cli

import (
	"errors"

	"github.com/spf13/cobra"
)

func newRatifyCmd() *cobra.Command {
	return &cobra.Command{
		Use:    "ratify <path>",
		Short:  "Add a clause to this shop's contract.",
		Hidden: true,
		RunE:   func(*cobra.Command, []string) error { return errors.New("not implemented") },
	}
}

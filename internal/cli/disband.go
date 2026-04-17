package cli

import (
	"errors"

	"github.com/spf13/cobra"
)

func newDisbandCmd() *cobra.Command {
	return &cobra.Command{
		Use:    "disband <dir>",
		Short:  "Unregister a shop.",
		Hidden: true,
		RunE:   func(*cobra.Command, []string) error { return errors.New("not implemented") },
	}
}

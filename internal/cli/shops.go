package cli

import (
	"errors"

	"github.com/spf13/cobra"
)

func newShopsCmd() *cobra.Command {
	return &cobra.Command{
		Use:    "shops",
		Short:  "List registered shops.",
		Hidden: true,
		RunE:   func(*cobra.Command, []string) error { return errors.New("not implemented") },
	}
}

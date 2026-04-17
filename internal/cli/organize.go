package cli

import (
	"errors"

	"github.com/spf13/cobra"
)

func newOrganizeCmd() *cobra.Command {
	return &cobra.Command{
		Use:    "organize [dir]",
		Short:  "Register a directory as an organized shop.",
		Hidden: true,
		RunE:   func(*cobra.Command, []string) error { return errors.New("not implemented") },
	}
}

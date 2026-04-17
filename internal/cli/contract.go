package cli

import (
	"errors"

	"github.com/spf13/cobra"
)

func newContractCmd() *cobra.Command {
	return &cobra.Command{
		Use:    "contract",
		Short:  "Show clauses present in this shop's contract.",
		Hidden: true,
		RunE:   func(*cobra.Command, []string) error { return errors.New("not implemented") },
	}
}

// Package cli wires up the union CLI with cobra.
package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

func newRootCmd() *cobra.Command {
	root := &cobra.Command{
		Use:           "union",
		Short:         "Composable, versioned AGENTS.md snippet management.",
		SilenceUsage:  true,
		SilenceErrors: true,
	}
	root.AddCommand(
		newInitCmd(),
		newNewCmd(),
		newClausesCmd(),
		newShowCmd(),
		newEditCmd(),
		newExpelCmd(),
		newOrganizeCmd(),
		newShopsCmd(),
		newDisbandCmd(),
		newRatifyCmd(),
		newStrikeCmd(),
		newContractCmd(),
	)
	return root
}

// Execute is the entry point called from main.
func Execute() {
	if err := newRootCmd().Execute(); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

package cli

import (
	"fmt"
	"os"

	"github.com/chazu/union/internal/shop"
	"github.com/spf13/cobra"
)

func newContractCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "contract",
		Short: "Show clauses currently ratified into this shop's contract.",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			_, contractPath, err := currentShop()
			if err != nil {
				return err
			}
			body, err := os.ReadFile(contractPath)
			if err != nil && !os.IsNotExist(err) {
				return err
			}
			blocks, err := shop.ParseContract(body)
			if err != nil {
				return err
			}
			if len(blocks) == 0 {
				fmt.Fprintln(cmd.OutOrStdout(), "(no clauses ratified)")
				return nil
			}
			for _, b := range blocks {
				fmt.Fprintln(cmd.OutOrStdout(), b.Path)
			}
			return nil
		},
	}
}

package cli

import (
	"fmt"

	"github.com/chazu/union/internal/paths"
	"github.com/spf13/cobra"
)

func newWhichCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "which",
		Short: "Print union paths (root, stores, shops file, current shop).",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			w := cmd.OutOrStdout()
			unionDir, err := paths.UnionDir()
			if err != nil {
				return err
			}
			fmt.Fprintf(w, "union dir:  %s\n", unionDir)
			fmt.Fprintf(w, "stores dir: %s\n", unionDir+"/"+paths.StoresSubdir)

			shopsFile, err := paths.ShopsFile()
			if err != nil {
				return err
			}
			fmt.Fprintf(w, "shops file: %s\n", shopsFile)

			s, contractPath, err := currentShop()
			if err == nil {
				fmt.Fprintf(w, "current shop: %s (contract: %s)\n", s.Dir, contractPath)
			}
			return nil
		},
	}
}

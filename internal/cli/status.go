package cli

import (
	"fmt"

	"github.com/chazu/union/internal/paths"
	"github.com/chazu/union/internal/shop"
	"github.com/chazu/union/internal/store"
	"github.com/spf13/cobra"
)

func newStatusCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Show global union status: stores, clauses, shops, current shop.",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			w := cmd.OutOrStdout()
			unionDir, err := paths.UnionDir()
			if err != nil {
				return err
			}
			stores, err := store.ListStores(unionDir)
			if err != nil {
				return err
			}

			totalClauses := 0
			for _, name := range stores {
				s, err := store.OpenNamed(unionDir, name)
				if err != nil {
					continue
				}
				cs, err := s.List("")
				if err != nil {
					continue
				}
				fmt.Fprintf(w, "store %-15s %d clauses\n", name, len(cs))
				totalClauses += len(cs)
			}
			if len(stores) == 0 {
				fmt.Fprintln(w, "no stores (run 'union init')")
				return nil
			}

			shopsPath, err := paths.ShopsFile()
			if err != nil {
				return err
			}
			r, err := shop.LoadRegistry(shopsPath)
			if err != nil {
				return err
			}
			shopList := r.List()
			fmt.Fprintf(w, "\n%d store(s), %d clause(s), %d shop(s)\n", len(stores), totalClauses, len(shopList))

			s, contractPath, err := currentShop()
			if err == nil {
				fmt.Fprintf(w, "current shop: %s (%s)\n", s.Dir, contractPath)
			}
			return nil
		},
	}
}

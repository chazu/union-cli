package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/chazu/union/internal/paths"
	"github.com/chazu/union/internal/shop"
	"github.com/chazu/union/internal/store"
	"github.com/spf13/cobra"
)

func newOrphansCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "orphans",
		Short: "List clauses not ratified by any shop.",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			w := cmd.OutOrStdout()
			unionDir, err := paths.UnionDir()
			if err != nil {
				return err
			}
			shopsPath, err := paths.ShopsFile()
			if err != nil {
				return err
			}
			r, err := shop.LoadRegistry(shopsPath)
			if err != nil {
				return err
			}

			ratified := map[string]bool{}
			for _, s := range r.List() {
				contractPath := filepath.Join(s.Dir, s.Contract)
				body, err := os.ReadFile(contractPath)
				if err != nil {
					continue
				}
				blocks, err := shop.ParseContract(body)
				if err != nil {
					fmt.Fprintf(w, "WARN %s: %v\n", contractPath, err)
					continue
				}
				for _, b := range blocks {
					ratified[b.Path] = true
				}
			}

			stores, err := store.ListStores(unionDir)
			if err != nil {
				return err
			}
			var count int
			for _, name := range stores {
				st, err := store.OpenNamed(unionDir, name)
				if err != nil {
					continue
				}
				clauses, err := st.List("")
				if err != nil {
					continue
				}
				for _, p := range clauses {
					qp := name + ":" + p
					if !ratified[qp] {
						fmt.Fprintln(w, qp)
						count++
					}
				}
			}
			if count == 0 {
				fmt.Fprintln(w, "no orphans")
			}
			return nil
		},
	}
}

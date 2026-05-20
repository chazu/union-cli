package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/chazu/union/internal/paths"
	"github.com/chazu/union/internal/qpath"
	"github.com/chazu/union/internal/shop"
	"github.com/chazu/union/internal/store"
	"github.com/spf13/cobra"
)

func newSyncCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "sync",
		Short: "Re-propagate all clauses to all shops (repair drift).",
		Long:  "Reads every shop's contract, fetches the current clause body from each store, and rewrites any blocks that have drifted. Leaves changes uncommitted for review.",
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

			var updated, skipped int
			for _, s := range r.List() {
				contractPath := filepath.Join(s.Dir, s.Contract)
				body, err := os.ReadFile(contractPath)
				if err != nil {
					if os.IsNotExist(err) {
						continue
					}
					return err
				}
				blocks, err := shop.ParseContract(body)
				if err != nil {
					fmt.Fprintf(w, "  skip %s: %v\n", contractPath, err)
					skipped++
					continue
				}
				changed := false
				current := body
				for _, b := range blocks {
					q, err := qpath.Parse(b.Path)
					if err != nil {
						fmt.Fprintf(w, "  warn %s: invalid path %q: %v\n", contractPath, b.Path, err)
						continue
					}
					st, err := store.OpenNamed(unionDir, q.Store)
					if err != nil {
						fmt.Fprintf(w, "  warn %s: store %q not found\n", contractPath, q.Store)
						continue
					}
					storeBody, err := st.Get(q.Path)
					if err != nil {
						fmt.Fprintf(w, "  warn %s: clause %s missing from store\n", contractPath, b.Path)
						continue
					}
					if string(b.Body) == string(storeBody) {
						continue
					}
					current, err = shop.UpdateClause(current, b.Path, storeBody)
					if err != nil {
						fmt.Fprintf(w, "  error %s %s: %v\n", contractPath, b.Path, err)
						continue
					}
					changed = true
				}
				if changed {
					if err := os.WriteFile(contractPath, current, 0o644); err != nil {
						return err
					}
					fmt.Fprintf(w, "  synced %s\n", contractPath)
					updated++
				}
			}
			if updated == 0 && skipped == 0 {
				fmt.Fprintln(w, "all shops already in sync")
			} else if updated > 0 {
				fmt.Fprintf(w, "%d shop(s) updated (changes uncommitted)\n", updated)
			}
			return nil
		},
	}
}

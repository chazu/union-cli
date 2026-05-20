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

func newVerifyCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "verify",
		Short: "Verify all shops' contracts match their store clauses.",
		Long:  "Checks every marked block in every shop's contract against the current store clause. Reports mismatches, missing clauses, and malformed markers. Exits non-zero if problems are found.",
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

			var problems int
			for _, s := range r.List() {
				contractPath := filepath.Join(s.Dir, s.Contract)
				body, err := os.ReadFile(contractPath)
				if err != nil {
					if os.IsNotExist(err) {
						fmt.Fprintf(w, "WARN %s: contract file missing\n", contractPath)
						problems++
						continue
					}
					return err
				}
				blocks, err := shop.ParseContract(body)
				if err != nil {
					fmt.Fprintf(w, "FAIL %s: malformed markers: %v\n", contractPath, err)
					problems++
					continue
				}
				for _, b := range blocks {
					q, err := qpath.Parse(b.Path)
					if err != nil {
						fmt.Fprintf(w, "FAIL %s: invalid path %q: %v\n", contractPath, b.Path, err)
						problems++
						continue
					}
					st, err := store.OpenNamed(unionDir, q.Store)
					if err != nil {
						fmt.Fprintf(w, "FAIL %s: store %q not found\n", contractPath, q.Store)
						problems++
						continue
					}
					storeBody, err := st.Get(q.Path)
					if err != nil {
						fmt.Fprintf(w, "FAIL %s: clause %s missing from store\n", contractPath, b.Path)
						problems++
						continue
					}
					if string(b.Body) != string(storeBody) {
						fmt.Fprintf(w, "DRIFT %s: %s differs from store\n", contractPath, b.Path)
						problems++
					}
				}
			}
			if problems > 0 {
				return fmt.Errorf("verification failed: %d problem(s) found", problems)
			}
			fmt.Fprintln(w, "OK: all contracts in sync")
			return nil
		},
	}
}

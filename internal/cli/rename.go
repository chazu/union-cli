package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/chazu/union/internal/paths"
	"github.com/chazu/union/internal/qpath"
	"github.com/chazu/union/internal/shop"
	"github.com/spf13/cobra"
)

func newRenameCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "rename <old-store:path> <new-store:path>",
		Short: "Rename a clause within the same store, rewriting all shop markers.",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			w := cmd.OutOrStdout()

			oldQ, err := qpath.Parse(args[0])
			if err != nil {
				return fmt.Errorf("old path: %w", err)
			}
			newQ, err := qpath.Parse(args[1])
			if err != nil {
				return fmt.Errorf("new path: %w", err)
			}
			if oldQ.Store != newQ.Store {
				return fmt.Errorf("rename must stay within the same store (got %s → %s)", oldQ.Store, newQ.Store)
			}

			s, err := openStoreFor(oldQ)
			if err != nil {
				return err
			}
			body, err := s.Get(oldQ.Path)
			if err != nil {
				return err
			}
			if s.Has(newQ.Path) {
				return fmt.Errorf("target clause already exists: %s", newQ)
			}

			if err := s.Put(newQ.Path, body, fmt.Sprintf("rename %s → %s", oldQ, newQ)); err != nil {
				return err
			}
			if err := s.Delete(oldQ.Path, fmt.Sprintf("rename %s → %s (remove old)", oldQ, newQ)); err != nil {
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
			for _, sh := range r.List() {
				contractPath := filepath.Join(sh.Dir, sh.Contract)
				contract, err := os.ReadFile(contractPath)
				if err != nil {
					if os.IsNotExist(err) {
						continue
					}
					return err
				}
				if !shop.HasClause(contract, oldQ.String()) {
					continue
				}
				updated, err := shop.RemoveClause(contract, oldQ.String())
				if err != nil {
					return fmt.Errorf("%s: remove old marker: %w", contractPath, err)
				}
				updated, err = shop.InsertClause(updated, newQ.String(), body)
				if err != nil {
					return fmt.Errorf("%s: insert new marker: %w", contractPath, err)
				}
				if err := os.WriteFile(contractPath, updated, 0o644); err != nil {
					return err
				}
				fmt.Fprintf(w, "  updated %s\n", contractPath)
			}
			fmt.Fprintf(w, "renamed %s → %s\n", oldQ, newQ)
			return nil
		},
	}
}

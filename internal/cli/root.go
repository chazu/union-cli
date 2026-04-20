// Package cli wires up the union CLI with cobra.
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
		newStoreCmd(),
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

func openStoreFor(q qpath.Qualified) (*store.Store, error) {
	unionDir, err := paths.UnionDir()
	if err != nil {
		return nil, err
	}
	s, err := store.OpenNamed(unionDir, q.Store)
	if err != nil {
		return nil, fmt.Errorf("no such store: %s (run 'union store list')", q.Store)
	}
	return s, nil
}

// currentShop returns the registered Shop for the current working directory,
// along with the absolute path to its contract file.
func currentShop() (shop.Shop, string, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return shop.Shop{}, "", err
	}
	cwd, err = resolveDir(cwd)
	if err != nil {
		return shop.Shop{}, "", err
	}
	shopsPath, err := paths.ShopsFile()
	if err != nil {
		return shop.Shop{}, "", err
	}
	r, err := shop.LoadRegistry(shopsPath)
	if err != nil {
		return shop.Shop{}, "", err
	}
	s, ok := r.Get(cwd)
	if !ok {
		return shop.Shop{}, "", fmt.Errorf("%s is not an organized shop. Run 'union organize' first.", cwd)
	}
	return s, filepath.Join(s.Dir, s.Contract), nil
}

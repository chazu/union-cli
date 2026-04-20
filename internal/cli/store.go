package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/chazu/union/internal/paths"
	"github.com/chazu/union/internal/qpath"
	"github.com/chazu/union/internal/shop"
	unionstore "github.com/chazu/union/internal/store"
	"github.com/spf13/cobra"
)

func newStoreCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "store",
		Short: "Manage clause stores (add, list, remove, remotes, push/pull).",
	}
	cmd.AddCommand(
		newStoreAddCmd(),
		newStoreListCmd(),
		newStoreRemoveCmd(),
		newStoreRemoteCmd(),
		newStorePushCmd(),
		newStorePullCmd(),
		newStoreFetchCmd(),
		newStoreStatusCmd(),
	)
	return cmd
}

func newStoreAddCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "add <name>",
		Short: "Create a new store at $UNION_DIR/stores/<name>.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]
			if err := qpath.ValidateStoreName(name); err != nil {
				return err
			}
			unionDir, err := paths.UnionDir()
			if err != nil {
				return err
			}
			dir := filepath.Join(unionDir, "stores", name)
			if _, err := os.Stat(dir); err == nil {
				return fmt.Errorf("store already exists: %s", name)
			}
			s, err := unionstore.InitNamed(unionDir, name)
			if err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "created store %q at %s\n", name, s.Root())
			return nil
		},
	}
}

func newStoreListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List stores.",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			unionDir, err := paths.UnionDir()
			if err != nil {
				return err
			}
			names, err := unionstore.ListStores(unionDir)
			if err != nil {
				return err
			}
			for _, n := range names {
				fmt.Fprintln(cmd.OutOrStdout(), n)
			}
			return nil
		},
	}
}

func newStoreRemoveCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "remove <name>",
		Short: "Delete a store (refused if clauses from it are ratified).",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, cmdArgs []string) error {
			name := cmdArgs[0]
			if err := qpath.ValidateStoreName(name); err != nil {
				return err
			}
			unionDir, err := paths.UnionDir()
			if err != nil {
				return err
			}
			dir := filepath.Join(unionDir, "stores", name)
			if _, err := os.Stat(dir); err != nil {
				return fmt.Errorf("no such store: %s", name)
			}
			offenders, err := shopsReferencingStore(name)
			if err != nil {
				return err
			}
			if len(offenders) > 0 {
				sort.Strings(offenders)
				return fmt.Errorf("refusing to remove store %q: still ratified in shop(s): %v", name, offenders)
			}
			if err := os.RemoveAll(dir); err != nil {
				return fmt.Errorf("remove store: %w", err)
			}
			return nil
		},
	}
}

// shopsReferencingStore returns shop dirs whose contracts hold any clause
// from the given store.
func shopsReferencingStore(storeName string) ([]string, error) {
	shopsPath, err := paths.ShopsFile()
	if err != nil {
		return nil, err
	}
	r, err := shop.LoadRegistry(shopsPath)
	if err != nil {
		return nil, err
	}
	var hits []string
	for _, s := range r.List() {
		contractPath := filepath.Join(s.Dir, s.Contract)
		body, err := os.ReadFile(contractPath)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return nil, err
		}
		blocks, err := shop.ParseContract(body)
		if err != nil {
			continue // malformed contract: don't block removal on it
		}
		for _, b := range blocks {
			q, err := qpath.Parse(b.Path)
			if err != nil {
				continue
			}
			if q.Store == storeName {
				hits = append(hits, s.Dir)
				break
			}
		}
	}
	return hits, nil
}

// Stubs filled in by Tasks 11 (remote) and 12 (push/pull/fetch/status).
func newStoreRemoteCmd() *cobra.Command { return &cobra.Command{Use: "remote", Short: "stub"} }
func newStorePushCmd() *cobra.Command   { return &cobra.Command{Use: "push", Short: "stub"} }
func newStorePullCmd() *cobra.Command   { return &cobra.Command{Use: "pull", Short: "stub"} }
func newStoreFetchCmd() *cobra.Command  { return &cobra.Command{Use: "fetch", Short: "stub"} }
func newStoreStatusCmd() *cobra.Command { return &cobra.Command{Use: "status", Short: "stub"} }

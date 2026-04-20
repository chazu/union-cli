package cli

import (
	"fmt"
	"strings"

	"github.com/chazu/union/internal/paths"
	"github.com/chazu/union/internal/store"
	"github.com/spf13/cobra"
)

func newClausesCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "clauses [<store>:<prefix>]",
		Short: "List clauses across stores (store:path form). Optional store:<prefix> filter.",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			unionDir, err := paths.UnionDir()
			if err != nil {
				return err
			}
			stores, err := store.ListStores(unionDir)
			if err != nil {
				return err
			}

			wantStore := ""
			wantPrefix := ""
			if len(args) == 1 {
				arg := args[0]
				i := strings.IndexByte(arg, ':')
				if i < 0 {
					return fmt.Errorf("filter must be qualified as <store>:<prefix> (got %q)", arg)
				}
				wantStore = arg[:i]
				wantPrefix = arg[i+1:]
			}

			for _, name := range stores {
				if wantStore != "" && name != wantStore {
					continue
				}
				s, err := store.OpenNamed(unionDir, name)
				if err != nil {
					return err
				}
				ps, err := s.List(wantPrefix)
				if err != nil {
					return err
				}
				for _, p := range ps {
					fmt.Fprintf(cmd.OutOrStdout(), "%s:%s\n", name, p)
				}
			}
			return nil
		},
	}
}
